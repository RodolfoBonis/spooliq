package helpers

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"plain", "plain"},
		{"50%", `50\%`},
		{"a_b", `a\_b`},
		{`back\slash`, `back\\slash`},
		{`%_\`, `\%\_\\`},
		{"", ""},
	}
	for _, tt := range tests {
		if got := EscapeLike(tt.in); got != tt.want {
			t.Errorf("EscapeLike(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
