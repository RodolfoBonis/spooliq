package parser

import "testing"

func TestParseDuration(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		want   int
		wantOK bool
	}{
		{"full dhms", "1d 2h 3m 4s", 86400 + 2*3600 + 3*60 + 4, true},
		{"hours minutes", "2h 5m", 2*3600 + 5*60, true},
		{"minutes seconds", "45m 10s", 45*60 + 10, true},
		{"single hour", "1h", 3600, true},
		{"plain seconds", "3600", 3600, true},
		{"plain seconds float", "3600.0", 3600, true},
		{"no spaces", "1h2m3s", 3600 + 120 + 3, true},
		{"with surrounding text trimmed", "  2h 30m  ", 2*3600 + 30*60, true},
		{"days only", "2d", 2 * 86400, true},
		{"empty", "", 0, false},
		{"garbage", "abc", 0, false},
		{"zero seconds", "0", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseDuration(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
