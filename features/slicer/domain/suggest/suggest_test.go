package suggest

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
)

func TestMatch_NoCandidates(t *testing.T) {
	if got := Match("#FF0000", "PLA", nil); got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestMatch_Exact(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "a", Name: "Red PLA", ColorHex: "#FF0000", Material: "Prusament PLA"},
		{FilamentID: "b", Name: "Green PLA", ColorHex: "#00FF00", Material: "PLA"},
	}
	s := Match("#FF0000", "PLA", cands)
	if s == nil {
		t.Fatal("expected a suggestion")
	}
	if s.FilamentID != "a" {
		t.Errorf("filament = %q, want a", s.FilamentID)
	}
	if s.Confidence != entities.ConfidenceExact {
		t.Errorf("confidence = %q, want exact", s.Confidence)
	}
	if s.Distance == nil || *s.Distance > 2.5 {
		t.Errorf("distance = %v, want < 2.5", s.Distance)
	}
}

func TestMatch_CloseWhenMaterialMismatchButColorNear(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "a", Name: "Red PETG", ColorHex: "#FF0000", Material: "PETG"},
	}
	s := Match("#FF0000", "PLA", cands)
	if s == nil {
		t.Fatal("expected a suggestion")
	}
	// Same color but wrong material -> cannot be exact; distance 0 < 15 -> close.
	if s.Confidence != entities.ConfidenceClose {
		t.Errorf("confidence = %q, want close", s.Confidence)
	}
}

func TestMatch_NoneWhenFar(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "a", Name: "Green PLA", ColorHex: "#00FF00", Material: "PLA"},
	}
	s := Match("#FF0000", "PLA", cands)
	if s == nil {
		t.Fatal("expected nearest suggestion")
	}
	if s.FilamentID != "a" {
		t.Errorf("filament = %q", s.FilamentID)
	}
	if s.Confidence != entities.ConfidenceNone {
		t.Errorf("confidence = %q, want none", s.Confidence)
	}
}

func TestMatch_PrefersSameMaterial(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "petg", Name: "Red PETG", ColorHex: "#FF0000", Material: "PETG"},  // perfect color, wrong material
		{FilamentID: "pla", Name: "Reddish PLA", ColorHex: "#FF3000", Material: "PLA"}, // near color, right material
	}
	s := Match("#FF0000", "PLA", cands)
	if s == nil {
		t.Fatal("expected a suggestion")
	}
	if s.FilamentID != "pla" {
		t.Errorf("filament = %q, want pla (same material preferred)", s.FilamentID)
	}
	if s.Confidence == entities.ConfidenceNone {
		t.Errorf("confidence = %q, want exact/close", s.Confidence)
	}
}

func TestMatch_FallsBackToNearestWhenSameMaterialFar(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "petg", Name: "Red PETG", ColorHex: "#FF0000", Material: "PETG"}, // perfect color
		{FilamentID: "pla", Name: "Blue PLA", ColorHex: "#0000FF", Material: "PLA"},   // far color, right material
	}
	s := Match("#FF0000", "PLA", cands)
	if s == nil {
		t.Fatal("expected a suggestion")
	}
	// Same-material (pla) is far (>=15), so the globally nearest (petg) is chosen.
	if s.FilamentID != "petg" {
		t.Errorf("filament = %q, want petg (nearest fallback)", s.FilamentID)
	}
}

func TestMatch_NoColorFallsBackToMaterial(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "a", Name: "Some PETG", ColorHex: "#112233", Material: "PETG"},
		{FilamentID: "b", Name: "Some PLA", ColorHex: "#445566", Material: "PLA"},
	}
	s := Match("", "PLA", cands)
	if s == nil {
		t.Fatal("expected a material-only suggestion")
	}
	if s.FilamentID != "b" {
		t.Errorf("filament = %q, want b", s.FilamentID)
	}
	if s.Confidence != entities.ConfidenceNone || s.Distance != nil {
		t.Errorf("want confidence none and nil distance, got %q / %v", s.Confidence, s.Distance)
	}
}

func TestMatch_NoColorNoMaterialMatch(t *testing.T) {
	cands := []Candidate{
		{FilamentID: "a", Name: "Some PETG", ColorHex: "#112233", Material: "PETG"},
	}
	if s := Match("", "PLA", cands); s != nil {
		t.Errorf("expected nil, got %+v", s)
	}
}
