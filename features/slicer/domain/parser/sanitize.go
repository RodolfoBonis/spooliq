package parser

import (
	"strings"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/color"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

// Output caps. Parsed files are untrusted: a hostile or buggy slicer could emit
// absurd counts or huge strings, which we must not return or persist unbounded.
const (
	maxPlates            = 64
	maxFilamentsPerPlate = 32
	maxWarnings          = 20
	maxMaterialLen       = 32
	maxSlicerFieldLen    = 64
	maxPlateNameLen      = 128
)

// sanitizeAnalysis clamps counts and string lengths and normalizes/validates each
// filament color to canonical uppercase "#RRGGBB" (invalid colors are dropped).
// It mutates a in place and guarantees a.Warnings is non-nil.
func sanitizeAnalysis(a *entities.Analysis) {
	if a == nil {
		return
	}

	a.Slicer.Name = truncateRunes(a.Slicer.Name, maxSlicerFieldLen)
	a.Slicer.Version = truncateRunes(a.Slicer.Version, maxSlicerFieldLen)

	if len(a.Plates) > maxPlates {
		a.Plates = a.Plates[:maxPlates]
	}
	for i := range a.Plates {
		p := &a.Plates[i]
		p.Name = truncateRunes(p.Name, maxPlateNameLen)
		if len(p.Filaments) > maxFilamentsPerPlate {
			p.Filaments = p.Filaments[:maxFilamentsPerPlate]
		}
		for j := range p.Filaments {
			f := &p.Filaments[j]
			if norm, ok := color.Normalize(f.ColorHex); ok {
				f.ColorHex = norm
			} else {
				f.ColorHex = ""
			}
			f.Material = truncateRunes(strings.TrimSpace(f.Material), maxMaterialLen)
		}
	}

	if len(a.Warnings) > maxWarnings {
		a.Warnings = a.Warnings[:maxWarnings]
	}
	if a.Warnings == nil {
		a.Warnings = []string{}
	}
}

// hasSlicingData reports whether an analysis carries any usable slicing data: at
// least one plate with a print time or any filament usage. Used to reject empty
// or non-sliced files with ErrNotSliced.
func hasSlicingData(a *entities.Analysis) bool {
	if a == nil {
		return false
	}
	for _, p := range a.Plates {
		if p.PrintTimeSeconds > 0 || len(p.Filaments) > 0 {
			return true
		}
	}
	return false
}

// truncateRunes trims s to at most n runes (not bytes) so multi-byte characters
// are never cut mid-rune.
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
