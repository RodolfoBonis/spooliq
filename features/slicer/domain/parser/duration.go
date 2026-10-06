package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// durationUnitRe matches a number followed by a d/h/m/s unit, e.g. "2h", "45m",
// "10s", "1d". Whitespace between the number and the unit is tolerated.
var durationUnitRe = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*([dhms])`)

// plainSecondsRe matches a bare integer (optionally a float) with no unit, used
// for Cura's `;TIME:3600` style value.
var plainSecondsRe = regexp.MustCompile(`^\s*\d+(?:\.\d+)?\s*$`)

// ParseDuration parses a human slicer duration into whole seconds. It handles:
//   - "1d 2h 3m 4s", "2h 5m", "45m 10s" (unit-suffixed, any subset/order);
//   - "3600" / "3600.0" (plain seconds).
//
// Returns (seconds, true) on success. Unrecognized or empty input yields
// (0, false).
func ParseDuration(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// Plain seconds (no unit letters).
	if plainSecondsRe.MatchString(s) {
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return 0, false
		}
		return int(f + 0.5), true
	}

	matches := durationUnitRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, false
	}

	var total float64
	for _, m := range matches {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(m[2]) {
		case "d":
			total += v * 86400
		case "h":
			total += v * 3600
		case "m":
			total += v * 60
		case "s":
			total += v
		}
	}
	if total <= 0 {
		return 0, false
	}
	return int(total + 0.5), true
}
