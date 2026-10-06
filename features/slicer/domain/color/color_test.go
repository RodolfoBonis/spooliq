package color

import (
	"math"
	"testing"
)

func TestParseHex(t *testing.T) {
	cases := []struct {
		in      string
		want    RGB
		wantErr bool
	}{
		{"#FF0000", RGB{255, 0, 0}, false},
		{"00FF00", RGB{0, 255, 0}, false},
		{"#f80", RGB{255, 136, 0}, false},
		{"  #0000FF  ", RGB{0, 0, 255}, false},
		{"", RGB{}, true},
		{"#12345", RGB{}, true},
		{"#GGGGGG", RGB{}, true},
	}
	for _, tc := range cases {
		got, err := ParseHex(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseHex(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseHex(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseHex(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestDistance_Identical(t *testing.T) {
	d, err := Distance("#AABBCC", "#aabbcc")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if d > 1e-9 {
		t.Errorf("distance of identical colors = %v, want 0", d)
	}
}

func TestDistance_Ordering(t *testing.T) {
	// A near-red should be much closer to red than green is.
	near, err := Distance("#FF0000", "#FE0000")
	if err != nil {
		t.Fatal(err)
	}
	far, err := Distance("#FF0000", "#00FF00")
	if err != nil {
		t.Fatal(err)
	}
	if near >= far {
		t.Errorf("near=%v should be < far=%v", near, far)
	}
	if near >= 2.5 {
		t.Errorf("near-identical distance = %v, expected < 2.5", near)
	}
}

func TestCIEDE2000_KnownPair(t *testing.T) {
	// Sharma reference-style sanity check: white vs black has a large L* delta.
	white := RGB{255, 255, 255}.ToLab()
	black := RGB{0, 0, 0}.ToLab()
	d := CIEDE2000(white, black)
	if d < 95 || d > 105 {
		t.Errorf("white-black CIEDE2000 = %v, want ~100", d)
	}
}

func TestInvalidHexDistance(t *testing.T) {
	if _, err := Distance("nope", "#FFFFFF"); err == nil {
		t.Error("expected error for invalid hex")
	}
}

func TestLabFinite(t *testing.T) {
	lab := RGB{10, 20, 30}.ToLab()
	if math.IsNaN(lab.L) || math.IsNaN(lab.A) || math.IsNaN(lab.B) {
		t.Errorf("Lab has NaN: %+v", lab)
	}
}
