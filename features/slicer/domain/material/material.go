// Package material normalizes free-form material names (from slicers and from the
// filament catalog) to a canonical token, and exposes filament densities used to
// estimate grams from length. It is pure and dependency-free.
package material

import (
	"regexp"
	"strings"
)

// Densities in g/cm³ keyed by canonical material token. Used by the Cura G-code
// path (which reports length, not weight) to estimate grams.
var densities = map[string]float64{
	"PLA":  1.24,
	"PLA+": 1.24,
	"PETG": 1.27,
	"ABS":  1.04,
	"ASA":  1.07,
	"TPU":  1.21,
	"PA":   1.14,
	"PC":   1.20,
}

// DefaultDensity is used when the material is unknown or absent.
const DefaultDensity = 1.24

// keywords are the recognized material tokens, ordered longest/most-specific
// first so e.g. "PLA+" wins over "PLA" and "PETG" over "PET"/"PA".
var keywords = []string{
	"PLA+", "PCTG", "PETG", "NYLON", "PVA", "HIPS",
	"PLA", "ABS", "ASA", "TPU", "PET", "PA6", "PA12", "PA", "PC", "PP",
}

// aliases maps a matched keyword to its canonical token.
var aliases = map[string]string{
	"NYLON": "PA",
	"PA6":   "PA",
	"PA12":  "PA",
}

// tokenRe matches a material keyword as a whole token (bounded by start/end or a
// non-alphanumeric character); '+' is allowed as part of the token (PLA+).
var tokenRe = buildTokenRe()

func buildTokenRe() *regexp.Regexp {
	parts := make([]string, len(keywords))
	for i, k := range keywords {
		parts[i] = regexp.QuoteMeta(k)
	}
	// (?:^|[^A-Z0-9])(KEYWORD)(?:$|[^A-Z0-9+]) over an uppercased string.
	return regexp.MustCompile(`(?:^|[^A-Z0-9])(` + strings.Join(parts, "|") + `)(?:$|[^A-Z0-9])`)
}

// Token returns the canonical material token for a free-form name (e.g.
// "Prusament PLA" -> "PLA", "Nylon CF" -> "PA"), or "" when none is recognized.
func Token(name string) string {
	up := strings.ToUpper(strings.TrimSpace(name))
	if up == "" {
		return ""
	}
	// Pad so a keyword at either boundary still has a non-alphanumeric neighbor.
	m := tokenRe.FindStringSubmatch(" " + up + " ")
	if m == nil {
		return ""
	}
	tok := m[1]
	if canon, ok := aliases[tok]; ok {
		return canon
	}
	return tok
}

// Matches reports whether two free-form material names resolve to the same
// canonical token. Returns false when either side is unrecognized.
func Matches(a, b string) bool {
	ta, tb := Token(a), Token(b)
	return ta != "" && ta == tb
}

// Density returns the filament density (g/cm³) for a free-form material name,
// falling back to DefaultDensity when unknown.
func Density(name string) float64 {
	if d, ok := densities[Token(name)]; ok {
		return d
	}
	return DefaultDensity
}
