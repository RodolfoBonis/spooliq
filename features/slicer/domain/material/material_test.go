package material

import "testing"

func TestToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"PLA", "PLA"},
		{"Prusament PLA", "PLA"},
		{"pla", "PLA"},
		{"PLA+", "PLA+"},
		{"Generic PETG", "PETG"},
		{"ABS", "ABS"},
		{"Polymaker ASA", "ASA"},
		{"TPU 95A", "TPU"},
		{"Nylon CF", "PA"},
		{"PA6-CF", "PA"},
		{"PC Blend", "PC"},
		{"Wood", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := Token(tc.in); got != tc.want {
			t.Errorf("Token(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMatches(t *testing.T) {
	if !Matches("PLA", "Prusament PLA") {
		t.Error("PLA should match Prusament PLA")
	}
	if Matches("PLA", "PETG") {
		t.Error("PLA should not match PETG")
	}
	if Matches("Wood", "Wood") {
		t.Error("unrecognized materials should not match")
	}
	if !Matches("Nylon", "PA6") {
		t.Error("Nylon should match PA6 via canonical PA")
	}
}

func TestDensity(t *testing.T) {
	if Density("PLA") != 1.24 {
		t.Errorf("PLA density = %v", Density("PLA"))
	}
	if Density("PETG") != 1.27 {
		t.Errorf("PETG density = %v", Density("PETG"))
	}
	if Density("Unknown") != DefaultDensity {
		t.Errorf("unknown density = %v, want default", Density("Unknown"))
	}
}
