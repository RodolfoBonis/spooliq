package insights

import (
	"fmt"
	"math"
	"strings"
)

// brl formats cents as pt-BR currency: 123456 -> "R$ 1.234,56".
func brl(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	reais := cents / 100
	digits := fmt.Sprintf("%d", reais)
	var grouped strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(d)
	}
	return fmt.Sprintf("%sR$ %s,%02d", sign, grouped.String(), cents%100)
}

// pct formats a percentage with one decimal and a comma: 12.34 -> "12,3%".
func pct(v float64) string {
	return strings.Replace(fmt.Sprintf("%.1f%%", v), ".", ",", 1)
}

// days formats hours as whole days: 80 -> "3 dias", 20 -> "1 dia".
func days(hours float64) string {
	d := int(math.Max(1, math.Round(hours/24)))
	if d == 1 {
		return "1 dia"
	}
	return fmt.Sprintf("%d dias", d)
}

// plural picks the singular or plural word for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// num formats a number with one decimal and a comma: 3.25 -> "3,3".
func num(v float64) string {
	return strings.Replace(fmt.Sprintf("%.1f", v), ".", ",", 1)
}
