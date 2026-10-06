// Package entities holds the slicer analysis output model. It has no gin, DB or
// slicer-internal dependencies so it can be shared by the pure parsers, the use
// cases and the model3d feature (which persists it as JSONB).
package entities

// Source values for Analysis.Source.
const (
	// SourceGCode marks an analysis produced from a plain .gcode file.
	SourceGCode = "gcode"
	// Source3MF marks an analysis produced from a .3mf / .gcode.3mf archive.
	Source3MF = "3mf"
)

// Confidence values for a filament Suggestion.
const (
	// ConfidenceExact — same material and color distance < 2.5.
	ConfidenceExact = "exact"
	// ConfidenceClose — color distance < 15.
	ConfidenceClose = "close"
	// ConfidenceNone — nearest available filament, but not a good match.
	ConfidenceNone = "none"
)

// Analysis is the full result of analyzing a sliced print file. It is also the
// shape persisted on a 3D model (there WITHOUT the per-filament suggestions,
// which are computed at read time).
type Analysis struct {
	// Source is "gcode" or "3mf".
	Source string `json:"source"`
	// Slicer identifies the program that produced the file (best-effort).
	Slicer Slicer `json:"slicer"`
	// Plates is one entry per build plate. Plain G-code yields a single plate.
	Plates []Plate `json:"plates"`
	// Warnings carries non-fatal, user-facing pt-BR notes. Never nil.
	Warnings []string `json:"warnings"`
}

// Slicer identifies the slicing program and its version.
type Slicer struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Plate is a single build plate with its print time and per-slot filament usage.
type Plate struct {
	// Index is the plate number (1-based for 3MF; 1 for plain G-code).
	Index int `json:"index"`
	// Name is an optional human label for the plate.
	Name string `json:"name,omitempty"`
	// PrintTimeSeconds is the estimated print time in seconds.
	PrintTimeSeconds int `json:"print_time_seconds"`
	// Estimated is true when the grams were derived (e.g. Cura, from length and
	// density) rather than reported directly by the slicer.
	Estimated bool `json:"estimated"`
	// Filaments is the per-slot usage. Slots with 0g are dropped.
	Filaments []Filament `json:"filaments"`
}

// Filament is the usage of one filament slot on a plate.
type Filament struct {
	// Slot is the 1-based extruder/filament slot number.
	Slot int `json:"slot"`
	// Grams is the filament weight used in this slot.
	Grams float64 `json:"grams"`
	// LengthMM is the filament length used, in millimeters, when known.
	LengthMM *float64 `json:"length_mm,omitempty"`
	// ColorHex is the slicer-reported color (e.g. "#FF0000"), when known.
	ColorHex string `json:"color_hex,omitempty"`
	// Material is the slicer-reported material (e.g. "PLA"), when known.
	Material string `json:"material,omitempty"`
	// Suggestion is the best matching org filament, or null when none applies.
	// It is always present as a key (null when there is no suggestion).
	Suggestion *Suggestion `json:"suggestion"`
}

// Suggestion is the best-matching organization filament for a slot.
type Suggestion struct {
	FilamentID string `json:"filament_id"`
	Name       string `json:"name"`
	ColorHex   string `json:"color_hex"`
	Material   string `json:"material"`
	// Confidence is "exact", "close" or "none".
	Confidence string `json:"confidence"`
	// Distance is the perceptual color distance to the slot color, when computed.
	Distance *float64 `json:"distance,omitempty"`
}
