// Package color provides pure color math used to match a slicer-reported filament
// color against the organization's filament catalog. It converts sRGB hex colors
// to CIE L*a*b* and computes perceptual distance with CIEDE2000 (with a CIE76
// fallback). Everything here is deterministic and dependency-free so it can be
// unit-tested in isolation.
package color

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrInvalidHex is returned when a string is not a parseable hex color.
var ErrInvalidHex = errors.New("invalid hex color")

// RGB is an 8-bit-per-channel sRGB color.
type RGB struct {
	R, G, B uint8
}

// Lab is a CIE L*a*b* color (D65 reference white).
type Lab struct {
	L, A, B float64
}

// ParseHex parses a hex color into an RGB, ignoring any alpha channel. A leading
// '#' is optional and parsing is case-insensitive. Accepted forms:
//
//	#RGB / RGB        (shorthand, each nibble doubled)
//	#RGBA / RGBA      (shorthand with alpha — alpha dropped)
//	#RRGGBB / RRGGBB
//	#RRGGBBAA / RRGGBBAA   (alpha dropped; emitted by Bambu/Orca)
//
// Any other form returns ErrInvalidHex.
func ParseHex(s string) (RGB, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 4:
		// Shorthand with alpha; expand RGB nibbles and drop the alpha nibble.
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
		// ok
	case 8:
		// RRGGBBAA; keep RGB, drop the AA alpha pair.
		s = s[:6]
	default:
		return RGB{}, ErrInvalidHex
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return RGB{}, ErrInvalidHex
	}
	return RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, nil
}

// Normalize parses any accepted hex form and returns it as canonical uppercase
// "#RRGGBB" (alpha dropped). ok is false when the input is not a valid hex color.
func Normalize(s string) (string, bool) {
	c, err := ParseHex(s)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B), true
}

// ToLab converts an sRGB color to CIE L*a*b* (D65).
func (c RGB) ToLab() Lab {
	// sRGB (0..1) -> linear RGB.
	lr := srgbToLinear(float64(c.R) / 255.0)
	lg := srgbToLinear(float64(c.G) / 255.0)
	lb := srgbToLinear(float64(c.B) / 255.0)

	// linear RGB -> XYZ (D65).
	x := lr*0.4124564 + lg*0.3575761 + lb*0.1804375
	y := lr*0.2126729 + lg*0.7151522 + lb*0.0721750
	z := lr*0.0193339 + lg*0.1191920 + lb*0.9503041

	// Normalize by D65 reference white.
	const (
		xn = 0.95047
		yn = 1.00000
		zn = 1.08883
	)
	fx := labF(x / xn)
	fy := labF(y / yn)
	fz := labF(z / zn)

	return Lab{
		L: 116*fy - 16,
		A: 500 * (fx - fy),
		B: 200 * (fy - fz),
	}
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Cbrt(t)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

// Distance returns the perceptual CIEDE2000 distance between two hex colors.
// Smaller is closer; 0 is identical. Returns ErrInvalidHex if either hex is
// unparseable.
func Distance(hex1, hex2 string) (float64, error) {
	c1, err := ParseHex(hex1)
	if err != nil {
		return 0, err
	}
	c2, err := ParseHex(hex2)
	if err != nil {
		return 0, err
	}
	return CIEDE2000(c1.ToLab(), c2.ToLab()), nil
}

// CIE76 is the simple Euclidean distance in Lab space. It is the documented
// fallback metric; CIEDE2000 is preferred.
func CIE76(a, b Lab) float64 {
	dl := a.L - b.L
	da := a.A - b.A
	db := a.B - b.B
	return math.Sqrt(dl*dl + da*da + db*db)
}

// CIEDE2000 computes the CIEDE2000 color-difference between two Lab colors.
// Implementation follows Sharma, Wu & Dalal (2005).
func CIEDE2000(lab1, lab2 Lab) float64 {
	const (
		kL = 1.0
		kC = 1.0
		kH = 1.0
	)

	l1, a1, b1 := lab1.L, lab1.A, lab1.B
	l2, a2, b2 := lab2.L, lab2.A, lab2.B

	c1 := math.Hypot(a1, b1)
	c2 := math.Hypot(a2, b2)
	cBar := (c1 + c2) / 2

	cBar7 := math.Pow(cBar, 7)
	g := 0.5 * (1 - math.Sqrt(cBar7/(cBar7+math.Pow(25, 7))))

	a1p := (1 + g) * a1
	a2p := (1 + g) * a2

	c1p := math.Hypot(a1p, b1)
	c2p := math.Hypot(a2p, b2)

	h1p := atan2Deg(b1, a1p)
	h2p := atan2Deg(b2, a2p)

	dLp := l2 - l1
	dCp := c2p - c1p

	var dhp float64
	switch {
	case c1p*c2p == 0:
		dhp = 0
	case math.Abs(h2p-h1p) <= 180:
		dhp = h2p - h1p
	case h2p-h1p > 180:
		dhp = h2p - h1p - 360
	default:
		dhp = h2p - h1p + 360
	}
	dHp := 2 * math.Sqrt(c1p*c2p) * math.Sin(deg2rad(dhp)/2)

	lBarp := (l1 + l2) / 2
	cBarp := (c1p + c2p) / 2

	var hBarp float64
	switch {
	case c1p*c2p == 0:
		hBarp = h1p + h2p
	case math.Abs(h1p-h2p) <= 180:
		hBarp = (h1p + h2p) / 2
	case h1p+h2p < 360:
		hBarp = (h1p + h2p + 360) / 2
	default:
		hBarp = (h1p + h2p - 360) / 2
	}

	t := 1 -
		0.17*math.Cos(deg2rad(hBarp-30)) +
		0.24*math.Cos(deg2rad(2*hBarp)) +
		0.32*math.Cos(deg2rad(3*hBarp+6)) -
		0.20*math.Cos(deg2rad(4*hBarp-63))

	dTheta := 30 * math.Exp(-math.Pow((hBarp-275)/25, 2))
	cBarp7 := math.Pow(cBarp, 7)
	rc := 2 * math.Sqrt(cBarp7/(cBarp7+math.Pow(25, 7)))
	rt := -rc * math.Sin(deg2rad(2*dTheta))

	lBarp50 := math.Pow(lBarp-50, 2)
	sl := 1 + (0.015*lBarp50)/math.Sqrt(20+lBarp50)
	sc := 1 + 0.045*cBarp
	sh := 1 + 0.015*cBarp*t

	termL := dLp / (kL * sl)
	termC := dCp / (kC * sc)
	termH := dHp / (kH * sh)

	return math.Sqrt(termL*termL + termC*termC + termH*termH + rt*termC*termH)
}

func atan2Deg(y, x float64) float64 {
	if y == 0 && x == 0 {
		return 0
	}
	d := rad2deg(math.Atan2(y, x))
	if d < 0 {
		d += 360
	}
	return d
}

func deg2rad(d float64) float64 { return d * math.Pi / 180 }
func rad2deg(r float64) float64 { return r * 180 / math.Pi }
