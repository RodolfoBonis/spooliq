package parser

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

// Safe-unzip limits guarding against malformed archives and zip bombs.
//
// Per-entry size and ratio caps apply to the entries we actually READ. Declared
// sizes of entries we never open (e.g. a 120MB Metadata/plate_1.gcode when
// slice_info.config already has the data) must not reject the archive.
const (
	maxZipEntries       = 2000
	maxTotalDeclared    = 2 << 30   // 2GB declared across all entries (sanity bound)
	maxEntrySize        = 100 << 20 // 100MB per fully-read entry (config files)
	maxCompressionRatio = 200       // uncompressed/compressed, above the floor
	ratioFloorBytes     = 1 << 20   // only enforce the ratio past 1MB uncompressed
	maxGCodeStreamBytes = 1 << 30   // decompression budget for a streamed embedded G-code
)

const (
	sliceInfoPath      = "Metadata/slice_info.config"
	modelPath          = "3D/3dmodel.model"
	modelHeadBytes     = 64 << 10 // the <metadata name="Application"> tag sits at the top
	projectSettingPath = "Metadata/project_settings.config"
	platePrefix        = "Metadata/plate_"
)

// Parse3MF analyzes a sliced 3MF (or .gcode.3mf) archive. It opens only the
// entries it needs and enforces entry-count, size and compression-ratio caps so a
// hostile archive cannot exhaust memory. Resolution order:
//  1. Metadata/slice_info.config (Bambu/Orca per-plate predictions and weights);
//  2. embedded Metadata/plate_N.gcode (parsed with the G-code parser);
//  3. Metadata/project_settings.config (JSON) only as a color/type fallback.
//
// A valid archive with no slicing data (e.g. a design-only PrusaSlicer 3MF)
// returns ErrNotSliced; a corrupt/zip-bomb archive returns ErrCorruptFile.
func Parse3MF(ra io.ReaderAt, size int64) (*entities.Analysis, error) {
	zr, err := zip.NewReader(ra, size)
	if err != nil {
		return nil, ErrCorruptFile
	}
	if len(zr.File) > maxZipEntries {
		return nil, ErrCorruptFile
	}

	var totalDeclared uint64
	index := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		totalDeclared += f.UncompressedSize64
		if totalDeclared > maxTotalDeclared {
			return nil, ErrCorruptFile
		}
		index[f.Name] = f
	}

	warnings := []string{}

	// Read the two small config entries ONCE and reuse the bytes for detection,
	// the color/type fallback and slice_info parsing.
	var sliceInfoData, projectSettingsData []byte
	if f, ok := index[sliceInfoPath]; ok {
		data, rerr := readZipEntry(f)
		if rerr != nil {
			return nil, rerr
		}
		sliceInfoData = data
	}
	if f, ok := index[projectSettingPath]; ok {
		if data, rerr := readZipEntry(f); rerr == nil {
			projectSettingsData = data
		}
	}

	fbColors, fbTypes := parseProjectSettings(projectSettingsData)
	slicer := detectSlicer3MF(readZipEntryHead(index[modelPath], modelHeadBytes), sliceInfoData, projectSettingsData)

	// 1. slice_info.config.
	if len(sliceInfoData) > 0 {
		plates := parseSliceInfo(sliceInfoData, fbColors, fbTypes)
		if len(plates) > 0 {
			a := &entities.Analysis{
				Source:   entities.Source3MF,
				Slicer:   slicer,
				Plates:   plates,
				Warnings: warnings,
			}
			sanitizeAnalysis(a)
			return a, nil
		}
	}

	// 2. Embedded plate_N.gcode files.
	if plates := parseEmbeddedGCode(index); len(plates) > 0 {
		a := &entities.Analysis{
			Source:   entities.Source3MF,
			Slicer:   slicer,
			Plates:   plates,
			Warnings: warnings,
		}
		sanitizeAnalysis(a)
		return a, nil
	}

	// 3. No slicing data at all.
	return nil, ErrNotSliced
}

// readZipEntry reads a single entry, enforcing the per-entry size cap and the
// compression-ratio (zip-bomb) guard against the ACTUAL decompressed bytes, not
// just the declared header values.
func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, ErrCorruptFile
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(io.LimitReader(rc, maxEntrySize+1))
	if err != nil {
		return nil, ErrCorruptFile
	}
	if int64(len(data)) > maxEntrySize {
		return nil, ErrCorruptFile
	}
	if f.CompressedSize64 > 0 && uint64(len(data)) > ratioFloorBytes {
		if float64(len(data))/float64(f.CompressedSize64) > maxCompressionRatio {
			return nil, ErrCorruptFile
		}
	}
	return data, nil
}

// --- slice_info.config (XML) ---

type sliceInfoConfig struct {
	XMLName xml.Name         `xml:"config"`
	Plates  []sliceInfoPlate `xml:"plate"`
}

type sliceInfoPlate struct {
	Metadata []sliceInfoKV `xml:"metadata"`
	Filament []struct {
		ID    string `xml:"id,attr"`
		Type  string `xml:"type,attr"`
		Color string `xml:"color,attr"`
		UsedM string `xml:"used_m,attr"`
		UsedG string `xml:"used_g,attr"`
	} `xml:"filament"`
}

type sliceInfoKV struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

func (p sliceInfoPlate) meta(key string) string {
	for _, m := range p.Metadata {
		if strings.EqualFold(m.Key, key) {
			return m.Value
		}
	}
	return ""
}

// parseSliceInfo builds plates from a slice_info.config document. Plates without
// any usable data (no time and no filament weight) are skipped. fbColors/fbTypes
// fill in missing per-slot color/type from project_settings.config.
func parseSliceInfo(data []byte, fbColors, fbTypes []string) []entities.Plate {
	var cfg sliceInfoConfig
	if err := xml.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	plates := make([]entities.Plate, 0, len(cfg.Plates))
	for _, sp := range cfg.Plates {
		idx := atoiDefault(sp.meta("index"), len(plates)+1)
		prediction, _ := ParseDuration(sp.meta("prediction"))

		filaments := make([]entities.Filament, 0, len(sp.Filament))
		for _, f := range sp.Filament {
			grams := parseFloatDefault(f.UsedG, 0)
			if grams <= 0 {
				continue
			}
			slot := atoiDefault(f.ID, len(filaments)+1)
			fil := entities.Filament{
				Slot:     slot,
				Grams:    round2(grams),
				ColorHex: normalizeHex(firstNonEmpty(f.Color, stringAt(fbColors, slot-1))),
				Material: firstNonEmpty(strings.TrimSpace(f.Type), stringAt(fbTypes, slot-1)),
			}
			if m := parseFloatDefault(f.UsedM, 0); m > 0 {
				lv := round2(m * 1000) // m -> mm
				fil.LengthMM = &lv
			}
			filaments = append(filaments, fil)
		}

		if prediction == 0 && len(filaments) == 0 {
			continue
		}
		plates = append(plates, entities.Plate{
			Index:            idx,
			PrintTimeSeconds: prediction,
			Estimated:        false,
			Filaments:        filaments,
		})
	}
	return plates
}

// --- embedded plate_N.gcode ---

func parseEmbeddedGCode(index map[string]*zip.File) []entities.Plate {
	type plateFile struct {
		n int
		f *zip.File
	}
	var files []plateFile
	for name, f := range index {
		if !strings.HasPrefix(name, platePrefix) || !strings.HasSuffix(strings.ToLower(name), ".gcode") {
			continue
		}
		numStr := strings.TrimSuffix(name[len(platePrefix):], ".gcode")
		numStr = strings.TrimSuffix(numStr, ".GCODE")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			n = len(files) + 1
		}
		files = append(files, plateFile{n: n, f: f})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].n < files[j].n })

	plates := make([]entities.Plate, 0, len(files))
	for _, pf := range files {
		// Real plate G-code is large (often >100MB for multi-color prints), so it is
		// streamed and only its head and tail (where slicers write metadata) are kept.
		data, err := readZipGCodeHeadTail(pf.f)
		if err != nil {
			continue
		}
		analysis, err := ParseGCode(bytes.NewReader(data), int64(len(data)))
		if err != nil || analysis == nil || len(analysis.Plates) == 0 {
			continue
		}
		plate := analysis.Plates[0]
		plate.Index = pf.n
		plates = append(plates, plate)
	}
	return plates
}

// readZipGCodeHeadTail streams an embedded G-code entry and returns its first and
// last headTailLimit bytes concatenated (separated by a newline), which is where
// slicers write their metadata comments. Memory stays bounded at ~2×headTailLimit
// regardless of the entry size; the decompression budget and the compression
// ratio are enforced on the bytes actually decompressed.
func readZipGCodeHeadTail(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, ErrCorruptFile
	}
	defer func() { _ = rc.Close() }()

	limit := int(headTailLimit)
	head := make([]byte, 0, limit)
	tail := make([]byte, limit) // ring buffer
	var tailLen, tailPos int
	var total uint64
	buf := make([]byte, 64<<10)
	for {
		n, rerr := rc.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			total += uint64(n)
			if total > maxGCodeStreamBytes {
				return nil, ErrCorruptFile
			}
			if f.CompressedSize64 > 0 && total > ratioFloorBytes &&
				float64(total)/float64(f.CompressedSize64) > maxCompressionRatio {
				return nil, ErrCorruptFile
			}
			if room := limit - len(head); room > 0 {
				take := min(room, len(chunk))
				head = append(head, chunk[:take]...)
				chunk = chunk[take:]
			}
			for len(chunk) > 0 {
				c := copy(tail[tailPos:], chunk)
				chunk = chunk[c:]
				tailPos = (tailPos + c) % limit
				tailLen = min(tailLen+c, limit)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return nil, ErrCorruptFile
		}
	}

	out := make([]byte, 0, len(head)+1+tailLen)
	out = append(out, head...)
	if tailLen > 0 {
		out = append(out, '\n')
		if tailLen < limit {
			out = append(out, tail[:tailLen]...)
		} else {
			out = append(out, tail[tailPos:]...)
			out = append(out, tail[:tailPos]...)
		}
	}
	return out, nil
}

// --- project_settings.config (JSON) fallback ---

func parseProjectSettings(data []byte) (colors, types []string) {
	if len(data) == 0 {
		return nil, nil
	}
	var settings struct {
		FilamentColour []string `json:"filament_colour"`
		FilamentType   []string `json:"filament_type"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, nil
	}
	return settings.FilamentColour, settings.FilamentType
}

// detectSlicer3MF is a best-effort slicer identification from the archive text.
var (
	appMetadataRe      = regexp.MustCompile(`<metadata name="Application">\s*([A-Za-z][A-Za-z ]*?)[- ]v?([0-9][0-9A-Za-z.+\-]*)\s*</metadata>`)
	bblClientVersionRe = regexp.MustCompile(`key="X-BBL-Client-Version"\s+value="([^"]+)"`)
)

// detectSlicer3MF identifies the slicer and its version. The most reliable source
// is the 3MF model's <metadata name="Application"> (e.g. "BambuStudio-02.08.02.61",
// "OrcaSlicer-2.1.1", "PrusaSlicer-2.7.1"); Bambu's slice_info X-BBL-Client-Version
// header and a text scan of the configs are fallbacks.
func detectSlicer3MF(modelHead, sliceInfoData, projectSettingsData []byte) entities.Slicer {
	if m := appMetadataRe.FindSubmatch(modelHead); m != nil {
		return entities.Slicer{Name: canonicalSlicerName(string(m[1])), Version: string(m[2])}
	}
	if m := bblClientVersionRe.FindSubmatch(sliceInfoData); m != nil {
		return entities.Slicer{Name: "BambuStudio", Version: string(m[1])}
	}
	var blob strings.Builder
	blob.Write(sliceInfoData)
	blob.Write(projectSettingsData)
	text := strings.ToLower(blob.String())
	switch {
	case strings.Contains(text, "bambustudio"):
		return entities.Slicer{Name: "BambuStudio"}
	case strings.Contains(text, "orcaslicer"):
		return entities.Slicer{Name: "OrcaSlicer"}
	case strings.Contains(text, "prusaslicer"):
		return entities.Slicer{Name: "PrusaSlicer"}
	default:
		return entities.Slicer{}
	}
}

// canonicalSlicerName maps an Application tag prefix to a display name.
func canonicalSlicerName(raw string) string {
	switch strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), " ", "")) {
	case "bambustudio":
		return "BambuStudio"
	case "orcaslicer":
		return "OrcaSlicer"
	case "prusaslicer":
		return "PrusaSlicer"
	case "superslicer":
		return "SuperSlicer"
	default:
		return strings.TrimSpace(raw)
	}
}

// readZipEntryHead returns up to n decompressed bytes from the start of an
// entry, or nil when the entry is missing or unreadable. Memory is bounded by n
// regardless of the entry size.
func readZipEntryHead(f *zip.File, n int64) []byte {
	if f == nil {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(io.LimitReader(rc, n))
	if err != nil {
		return nil
	}
	return data
}

// --- tiny helpers ---

func atoiDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseFloatDefault(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return def
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
