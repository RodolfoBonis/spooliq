// Package suggest selects the best-matching organization filament for a
// slicer-reported slot (material + color). It is pure and dependency-free so it
// can be unit-tested exhaustively.
package suggest

import (
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/color"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/material"
)

// Candidate is one filament from the organization catalog, reduced to the fields
// needed for matching.
type Candidate struct {
	FilamentID string
	Name       string
	ColorHex   string
	Material   string // free-form material name (e.g. "Prusament PLA")
}

// Thresholds for confidence buckets, in CIEDE2000 distance units.
const (
	exactMaxDistance = 2.5
	closeMaxDistance = 15.0
)

// Match returns the best org filament for a slot, or nil when there are no
// candidates (or nothing can be matched).
//
// Selection prefers the same material: when at least one same-material candidate
// is within the "close" threshold it is chosen; otherwise the globally nearest
// candidate by color is chosen. Confidence is then:
//   - "exact" when the material matches AND distance < 2.5;
//   - "close" when distance < 15;
//   - "none"  otherwise (still the nearest available filament).
//
// When the slot has no usable color, matching falls back to the first
// same-material candidate with confidence "none" and no distance.
func Match(slotColorHex, slotMaterial string, candidates []Candidate) *entities.Suggestion {
	if len(candidates) == 0 {
		return nil
	}

	_, slotColorErr := color.ParseHex(slotColorHex)
	hasColor := slotColorErr == nil

	if !hasColor {
		return matchWithoutColor(slotMaterial, candidates)
	}

	nearestSame := nearest(slotColorHex, candidates, slotMaterial)
	nearestAll := nearest(slotColorHex, candidates, "")

	chosen := nearestAll
	if nearestSame != nil && nearestSame.distance < closeMaxDistance {
		chosen = nearestSame
	}
	if chosen == nil {
		// No candidate had a parseable color; fall back to material-only.
		return matchWithoutColor(slotMaterial, candidates)
	}

	materialMatches := material.Matches(slotMaterial, chosen.cand.Material)
	confidence := entities.ConfidenceNone
	switch {
	case materialMatches && chosen.distance < exactMaxDistance:
		confidence = entities.ConfidenceExact
	case chosen.distance < closeMaxDistance:
		confidence = entities.ConfidenceClose
	}

	d := chosen.distance
	return &entities.Suggestion{
		FilamentID: chosen.cand.FilamentID,
		Name:       chosen.cand.Name,
		ColorHex:   chosen.cand.ColorHex,
		Material:   chosen.cand.Material,
		Confidence: confidence,
		Distance:   &d,
	}
}

// matchWithoutColor returns the first same-material candidate (confidence none,
// no distance), or nil when the slot material matches nothing.
func matchWithoutColor(slotMaterial string, candidates []Candidate) *entities.Suggestion {
	for i := range candidates {
		if material.Matches(slotMaterial, candidates[i].Material) {
			c := candidates[i]
			return &entities.Suggestion{
				FilamentID: c.FilamentID,
				Name:       c.Name,
				ColorHex:   c.ColorHex,
				Material:   c.Material,
				Confidence: entities.ConfidenceNone,
			}
		}
	}
	return nil
}

type scored struct {
	cand     Candidate
	distance float64
}

// nearest returns the candidate with the smallest color distance to slotColorHex.
// When materialFilter is non-empty, only candidates whose material matches it are
// considered. Candidates with an unparseable color are skipped. Returns nil when
// nothing qualifies.
func nearest(slotColorHex string, candidates []Candidate, materialFilter string) *scored {
	var best *scored
	for i := range candidates {
		c := candidates[i]
		if materialFilter != "" && !material.Matches(materialFilter, c.Material) {
			continue
		}
		d, err := color.Distance(slotColorHex, c.ColorHex)
		if err != nil {
			continue
		}
		if best == nil || d < best.distance {
			best = &scored{cand: c, distance: d}
		}
	}
	return best
}
