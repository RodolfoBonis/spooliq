package parser

import (
	"io"
	"strconv"
	"strings"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

// ParseGCode extracts print metadata from a .gcode file by reading ONLY comment
// lines. It scans the first headTailLimit bytes and the last headTailLimit bytes
// (via SectionReader on ra) because slicers differ in where they place summary
// metadata: Cura writes TIME at the top, PrusaSlicer/Orca write the filament and
// time summary at the very end. The middle (the actual moves) is never read.
func ParseGCode(ra io.ReaderAt, size int64) (*entities.Analysis, error) {
	g := &gcodeGatherer{}

	// Head.
	headLen := size
	if headLen > headTailLimit {
		headLen = headTailLimit
	}
	scanComments(io.NewSectionReader(ra, 0, headLen), g.process)

	// Tail (only when it does not overlap the head).
	if size > headTailLimit*2 {
		scanComments(io.NewSectionReader(ra, size-headTailLimit, headTailLimit), g.process)
	} else if size > headTailLimit {
		// File fits in [0, size) but is larger than the head window: scan the
		// remainder as a tail so nothing is missed and nothing is read twice.
		scanComments(io.NewSectionReader(ra, headLen, size-headLen), g.process)
	}

	return g.build(), nil
}

// gcodeGatherer accumulates metadata seen across the head and tail passes. Later
// occurrences do not overwrite earlier non-empty values (first wins), except for
// arrays which are only set once.
type gcodeGatherer struct {
	slicerName    string
	slicerVersion string

	primaryTimeSec int // PrusaSlicer "estimated printing time" / Orca "total estimated time"
	modelTimeSec   int // Orca "model printing time"
	curaTimeSec    int // Cura ";TIME:"

	grams   []float64
	lengths []float64 // millimeters
	colors  []string
	types   []string
}

func (g *gcodeGatherer) process(line string) bool {
	body := strings.TrimSpace(strings.TrimLeft(line, ";"))
	if body == "" {
		return true
	}
	lower := strings.ToLower(body)

	// --- Slicer identification ---
	g.detectSlicer(body, lower)

	// --- Time --- (time values may be followed by another ';'-separated label,
	// so stop at ';' when extracting them).
	switch {
	case strings.Contains(lower, "total estimated time"):
		if v, ok := ParseDuration(labeledValue(body, lower, "total estimated time", ':', true)); ok {
			g.primaryTimeSec = v
		}
		if strings.Contains(lower, "model printing time") {
			if v, ok := ParseDuration(labeledValue(body, lower, "model printing time", ':', true)); ok && g.modelTimeSec == 0 {
				g.modelTimeSec = v
			}
		}
	case strings.Contains(lower, "estimated printing time"):
		// PrusaSlicer/SuperSlicer: "estimated printing time (normal mode) = ..."
		if v, ok := ParseDuration(labeledValue(body, lower, "estimated printing time", '=', true)); ok && g.primaryTimeSec == 0 {
			g.primaryTimeSec = v
		}
	case strings.Contains(lower, "model printing time"):
		if v, ok := ParseDuration(labeledValue(body, lower, "model printing time", ':', true)); ok && g.modelTimeSec == 0 {
			g.modelTimeSec = v
		}
	case strings.HasPrefix(lower, "time:"):
		// Cura: ";TIME:3600" (seconds). Excludes ";TIME_ELAPSED:".
		if v, ok := ParseDuration(strings.TrimSpace(body[len("time:"):])); ok && g.curaTimeSec == 0 {
			g.curaTimeSec = v
		}
	}

	// --- Filament weight / length / color / type --- (colour/type values are
	// themselves ';'-separated lists, so do NOT stop at ';' for them).
	switch {
	case strings.Contains(lower, "filament used [g]"):
		if g.grams == nil {
			g.grams = parseFloatList(labeledValue(body, lower, "filament used [g]", '=', false), "")
		}
	case strings.Contains(lower, "total filament weight [g]"):
		if g.grams == nil {
			g.grams = parseFloatList(labeledValue(body, lower, "total filament weight [g]", ':', false), "")
		}
	case strings.Contains(lower, "filament used [mm]"):
		if g.lengths == nil {
			g.lengths = parseFloatList(labeledValue(body, lower, "filament used [mm]", '=', false), "")
		}
	case strings.Contains(lower, "filament used [cm3]"):
		// ignore volume; handled via grams/length
	case strings.HasPrefix(lower, "filament used:"):
		// Cura: ";Filament used: 1.23456m" (meters, possibly comma-separated).
		if g.lengths == nil {
			meters := parseFloatList(strings.TrimSpace(body[len("filament used:"):]), "m")
			g.lengths = make([]float64, len(meters))
			for i, m := range meters {
				g.lengths[i] = m * 1000 // m -> mm
			}
		}
	}

	if strings.Contains(lower, "filament_colour") || strings.Contains(lower, "filament_color") {
		if g.colors == nil {
			g.colors = splitList(labeledValue(body, lower, "filament_colour", '=', false), ';')
			if g.colors == nil {
				g.colors = splitList(labeledValue(body, lower, "filament_color", '=', false), ';')
			}
		}
	}
	if strings.Contains(lower, "filament_type") {
		if g.types == nil {
			g.types = splitList(labeledValue(body, lower, "filament_type", '=', false), ';')
		}
	}

	return true
}

func (g *gcodeGatherer) detectSlicer(body, lower string) {
	// set records a name/version, never overwriting an already-known value. This
	// lets e.g. Cura's ";FLAVOR:" set the name first and the later
	// ";Generated with Cura_SteamEngine x.y" line still fill in the version.
	set := func(name, version string) {
		if g.slicerName == "" {
			g.slicerName = name
		}
		if version != "" && g.slicerVersion == "" {
			g.slicerVersion = version
		}
	}
	switch {
	case strings.Contains(lower, "prusaslicer"):
		set("PrusaSlicer", versionAfter(body, "PrusaSlicer"))
	case strings.Contains(lower, "superslicer"):
		set("SuperSlicer", versionAfter(body, "SuperSlicer"))
	case strings.Contains(lower, "orcaslicer"):
		set("OrcaSlicer", versionAfter(body, "OrcaSlicer"))
	case strings.Contains(lower, "bambustudio"):
		set("BambuStudio", versionAfter(body, "BambuStudio"))
	case strings.Contains(lower, "cura_steamengine"):
		set("Cura", versionAfter(body, "Cura_SteamEngine"))
	case strings.HasPrefix(lower, "flavor:"):
		set("Cura", "")
	}
}

// build assembles the single-plate Analysis from the gathered fields.
func (g *gcodeGatherer) build() *entities.Analysis {
	warnings := []string{}

	// Resolve print time, preferring the total estimate (includes prep/heatup).
	printTime := g.primaryTimeSec
	if printTime == 0 {
		printTime = g.curaTimeSec
	}
	if printTime == 0 {
		printTime = g.modelTimeSec
	}
	if printTime == 0 {
		warnings = append(warnings, "Tempo de impressão não encontrado no arquivo.")
	}

	// Resolve grams, estimating from length when the slicer reported only length.
	grams := g.grams
	estimated := false
	if len(grams) == 0 && len(g.lengths) > 0 {
		grams = make([]float64, len(g.lengths))
		for i, l := range g.lengths {
			grams[i] = estimateGramsFromLengthMM(l, defaultDiameterMM, typeAt(g.types, i))
		}
		estimated = true
		warnings = append(warnings, "Pesos de filamento estimados a partir do comprimento (o fatiador não informou gramas).")
	}

	slots := maxLen(grams, g.lengths, g.colors, g.types)
	filaments := make([]entities.Filament, 0, slots)
	for i := 0; i < slots; i++ {
		grm := floatAt(grams, i)
		if grm <= 0 {
			continue // drop 0g slots
		}
		fil := entities.Filament{
			Slot:     i + 1,
			Grams:    round2(grm),
			ColorHex: normalizeHex(stringAt(g.colors, i)),
			Material: strings.TrimSpace(typeAt(g.types, i)),
		}
		if l := floatAt(g.lengths, i); l > 0 {
			lv := round2(l)
			fil.LengthMM = &lv
		}
		filaments = append(filaments, fil)
	}

	return &entities.Analysis{
		Source: entities.SourceGCode,
		Slicer: entities.Slicer{Name: g.slicerName, Version: g.slicerVersion},
		Plates: []entities.Plate{{
			Index:            1,
			PrintTimeSeconds: printTime,
			Estimated:        estimated,
			Filaments:        filaments,
		}},
		Warnings: warnings,
	}
}

// --- small parsing helpers (pure) ---

// labeledValue finds label (case-insensitive, using lower as the lowercased body)
// then the first occurrence of sep after it, and returns the following text,
// trimmed. When stopAtSemicolon is true the value is cut at the next ';' (used for
// time lines that chain multiple ';'-separated labels); when false the whole
// remainder is returned (used for ';'-separated colour/type LISTS).
func labeledValue(body, lower, label string, sep byte, stopAtSemicolon bool) string {
	idx := strings.Index(lower, strings.ToLower(label))
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(label):]
	sepIdx := strings.IndexByte(rest, sep)
	if sepIdx < 0 {
		return ""
	}
	val := rest[sepIdx+1:]
	if stopAtSemicolon {
		if end := strings.IndexByte(val, ';'); end >= 0 {
			val = val[:end]
		}
	}
	return strings.TrimSpace(val)
}

// parseFloatList splits a comma-separated list of numbers (each optionally
// suffixed by unitSuffix, e.g. "m") into floats. Unparseable entries become 0.
func parseFloatList(s, unitSuffix string) []float64 {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if unitSuffix != "" {
			p = strings.TrimSuffix(p, unitSuffix)
			p = strings.TrimSpace(p)
		}
		f, err := strconv.ParseFloat(p, 64)
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, f)
	}
	return out
}

// splitList splits by sep and trims; returns nil for empty input.
func splitList(s string, sep byte) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, string(sep))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// versionAfter returns the first version-looking token after keyword in s.
func versionAfter(s, keyword string) string {
	lower := strings.ToLower(s)
	idx := strings.Index(lower, strings.ToLower(keyword))
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(s[idx+len(keyword):])
	fields := strings.Fields(rest)
	for _, f := range fields {
		f = strings.TrimPrefix(f, "v")
		if f != "" && (f[0] >= '0' && f[0] <= '9') {
			return f
		}
	}
	return ""
}

func normalizeHex(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.HasPrefix(s, "#") {
		s = "#" + s
	}
	return s
}

func typeAt(arr []string, i int) string {
	if i < len(arr) {
		return arr[i]
	}
	return ""
}
func stringAt(arr []string, i int) string { return typeAt(arr, i) }

func floatAt(arr []float64, i int) float64 {
	if i < len(arr) {
		return arr[i]
	}
	return 0
}

func maxLen(a []float64, b []float64, c []string, d []string) int {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	if len(c) > m {
		m = len(c)
	}
	if len(d) > m {
		m = len(d)
	}
	return m
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
