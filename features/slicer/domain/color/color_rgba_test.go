package color

import "testing"

func TestParseHex_RGBA(t *testing.T) {
	cases := []struct {
		in   string
		want RGB
	}{
		{"#FF0000FF", RGB{255, 0, 0}},     // Bambu/Orca opaque red
		{"#00AE42FF", RGB{0, 0xAE, 0x42}}, // Bambu "Bambu Green"
		{"00ae42ff", RGB{0, 0xAE, 0x42}},  // no '#', lowercase
		{"#FF000080", RGB{255, 0, 0}},     // alpha dropped regardless of value
		{"#f80f", RGB{255, 136, 0}},       // 4-digit shorthand with alpha
	}
	for _, tc := range cases {
		got, err := ParseHex(tc.in)
		if err != nil {
			t.Errorf("ParseHex(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseHex(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"#00AE42FF", "#00AE42", true},
		{"#ff0000ff", "#FF0000", true},
		{"00ae42", "#00AE42", true},
		{"#f80", "#FF8800", true},
		{"#f80f", "#FF8800", true},
		{"not-a-color", "", false},
		{"", "", false},
		{"#12345", "", false},
	}
	for _, tc := range cases {
		got, ok := Normalize(tc.in)
		if ok != tc.wantOK {
			t.Errorf("Normalize(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDistance_IgnoresAlpha(t *testing.T) {
	d, err := Distance("#FF0000FF", "#FF0000")
	if err != nil {
		t.Fatal(err)
	}
	if d > 1e-9 {
		t.Errorf("distance ignoring alpha = %v, want 0", d)
	}
}
