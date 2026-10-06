package parser

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

// Safe-unzip limits guarding against malformed archives and zip bombs.
const (
	maxZipEntries        = 2000
	maxTotalUncompressed = 500 << 20 // 500MB total
	maxEntrySize         = 100 << 20 // 100MB per entry
	maxCompressionRatio  = 200       // uncompressed/compressed, above the floor
	ratioFloorBytes      = 1 << 20   // only enforce the ratio past 1MB uncompressed
)

const (
	sliceInfoPath      = "Metadata/slice_info.config"
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
		if f.UncompressedSize64 > maxEntrySize {
			return nil, ErrCorruptFile
		}
		if f.CompressedSize64 > 0 && f.UncompressedSize64 > ratioFloorBytes {
			if float64(f.UncompressedSize64)/float64(f.CompressedSize64) > maxCompressionRatio {
				return nil, ErrCorruptFile
			}
		}
		totalDeclared += f.UncompressedSize64
		if totalDeclared > maxTotalUncompressed {
			return nil, ErrCorruptFile
		}
		index[f.Name] = f
	}

	warnings := []string{}

	// Color/type fallback from project_settings.config (best-effort).
	fbColors, fbTypes := readProjectSettings(index)
	slicer := detectSlicer3MF(index)

	// 1. slice_info.config.
	if f, ok := index[sliceInfoPath]; ok {
		data, rerr := readZipEntry(f)
		if rerr != nil {
			return nil, rerr
		}
		plates := parseSliceInfo(data, fbColors, fbTypes)
		if len(plates) > 0 {
			return &entities.Analysis{
				Source:   entities.Source3MF,
				Slicer:   slicer,
				Plates:   plates,
				Warnings: warnings,
			}, nil
		}
	}

	// 2. Embedded plate_N.gcode files.
	if plates := parseEmbeddedGCode(index); len(plates) > 0 {
		return &entities.Analysis{
			Source:   entities.Source3MF,
			Slicer:   slicer,
			Plates:   plates,
			Warnings: warnings,
		}, nil
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
		data, err := readZipEntry(pf.f)
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

// --- project_settings.config (JSON) fallback ---

func readProjectSettings(index map[string]*zip.File) (colors, types []string) {
	f, ok := index[projectSettingPath]
	if !ok {
		return nil, nil
	}
	data, err := readZipEntry(f)
	if err != nil {
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
func detectSlicer3MF(index map[string]*zip.File) entities.Slicer {
	var blob strings.Builder
	for _, name := range []string{sliceInfoPath, projectSettingPath} {
		if f, ok := index[name]; ok {
			if data, err := readZipEntry(f); err == nil {
				blob.Write(data)
			}
		}
	}
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
