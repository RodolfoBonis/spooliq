package entities

import (
	"encoding/json"
	"fmt"
)

// ColorType defines the type of color configuration for a filament
type ColorType string

const (
	// ColorTypeSolid represents a solid color configuration
	ColorTypeSolid ColorType = "solid"
	// ColorTypeGradient represents a gradient color configuration
	ColorTypeGradient ColorType = "gradient"
	// ColorTypeDuo represents a duo color configuration with patterns
	ColorTypeDuo ColorType = "duo"
	// ColorTypeRainbow represents a rainbow color configuration
	ColorTypeRainbow ColorType = "rainbow"
	// ColorTypeMetallic represents a metallic color configuration
	ColorTypeMetallic ColorType = "metallic"
	// ColorTypeTransparent represents a transparent/translucent color configuration
	ColorTypeTransparent ColorType = "transparent"
	// ColorTypeWoodFill represents a wood-fill material color
	ColorTypeWoodFill ColorType = "wood-fill"
	// ColorTypeCarbonFiber represents a carbon fiber material color
	ColorTypeCarbonFiber ColorType = "carbon-fiber"
)

// IsValid checks if the color type is valid
func (ct ColorType) IsValid() bool {
	switch ct {
	case ColorTypeSolid, ColorTypeGradient, ColorTypeDuo, ColorTypeRainbow,
		ColorTypeMetallic, ColorTypeTransparent, ColorTypeWoodFill, ColorTypeCarbonFiber:
		return true
	default:
		return false
	}
}

// String returns the string representation of the color type
func (ct ColorType) String() string {
	return string(ct)
}

// ColorData represents the base interface for all color data types
type ColorData interface {
	GetType() ColorType
	GenerateCSS() string
	Validate() error
}

// SolidColorData represents a solid color configuration
type SolidColorData struct {
	Color string `json:"color" validate:"required,hexcolor"`
}

// GetType returns the color type for solid color data
func (s *SolidColorData) GetType() ColorType {
	return ColorTypeSolid
}

// GenerateCSS generates CSS color representation for solid color
func (s *SolidColorData) GenerateCSS() string {
	return s.Color
}

// Validate validates the solid color data
func (s *SolidColorData) Validate() error {
	if s.Color == "" {
		return fmt.Errorf("color cannot be empty")
	}
	return nil
}

// GradientStop represents a color stop in a gradient
type GradientStop struct {
	Color    string  `json:"color" validate:"required,hexcolor"`
	Position float64 `json:"position" validate:"required,min=0,max=100"`
}

// GradientColorData represents a gradient color configuration
type GradientColorData struct {
	Direction string         `json:"direction" validate:"required"`
	Colors    []GradientStop `json:"colors" validate:"required,min=2"`
}

// GetType returns the color type for gradient color data
func (g *GradientColorData) GetType() ColorType {
	return ColorTypeGradient
}

// GenerateCSS generates CSS gradient representation
func (g *GradientColorData) GenerateCSS() string {
	if len(g.Colors) == 0 {
		return ""
	}

	css := fmt.Sprintf("linear-gradient(%s", g.Direction)
	for _, stop := range g.Colors {
		css += fmt.Sprintf(", %s %.1f%%", stop.Color, stop.Position)
	}
	css += ")"
	return css
}

// Validate validates the gradient color data
func (g *GradientColorData) Validate() error {
	if g.Direction == "" {
		return fmt.Errorf("direction cannot be empty")
	}
	if len(g.Colors) < 2 {
		return fmt.Errorf("gradient must have at least 2 colors")
	}
	for i, stop := range g.Colors {
		if stop.Color == "" {
			return fmt.Errorf("color at position %d cannot be empty", i)
		}
		if stop.Position < 0 || stop.Position > 100 {
			return fmt.Errorf("position at %d must be between 0 and 100", i)
		}
	}
	return nil
}

// DuoPattern represents the pattern type for duo colors
type DuoPattern string

const (
	// DuoPatternStripes represents a striped duo color pattern
	DuoPatternStripes DuoPattern = "stripes"
	// DuoPatternSpots represents a spotted duo color pattern
	DuoPatternSpots DuoPattern = "spots"
	// DuoPatternRandom represents a random duo color pattern
	DuoPatternRandom DuoPattern = "random"
	// DuoPatternMarbled represents a marbled duo color pattern
	DuoPatternMarbled DuoPattern = "marbled"
)

// DuoColorData represents a duo color configuration
type DuoColorData struct {
	Primary   string     `json:"primary" validate:"required,hexcolor"`
	Secondary string     `json:"secondary" validate:"required,hexcolor"`
	Pattern   DuoPattern `json:"pattern" validate:"required"`
	Ratio     float64    `json:"ratio" validate:"min=0.1,max=0.9"` // Primary color ratio (0.1-0.9)
}

// GetType returns the color type for duo color data
func (d *DuoColorData) GetType() ColorType {
	return ColorTypeDuo
}

// GenerateCSS generates CSS representation for duo colors
func (d *DuoColorData) GenerateCSS() string {
	ratio := d.Ratio
	if ratio == 0 {
		ratio = 0.5 // Default 50/50
	}
	primaryPercent := ratio * 100
	secondaryPercent := (1 - ratio) * 100

	switch d.Pattern {
	case DuoPatternStripes:
		return fmt.Sprintf("linear-gradient(90deg, %s %.1f%%, %s %.1f%%)",
			d.Primary, primaryPercent, d.Secondary, secondaryPercent)
	case DuoPatternSpots:
		return fmt.Sprintf("radial-gradient(circle, %s 30%%, %s 30%%)", d.Primary, d.Secondary)
	case DuoPatternMarbled:
		return fmt.Sprintf("linear-gradient(45deg, %s 0%%, %s 25%%, %s 50%%, %s 75%%, %s 100%%)",
			d.Primary, d.Secondary, d.Primary, d.Secondary, d.Primary)
	default:
		return fmt.Sprintf("linear-gradient(45deg, %s %.1f%%, %s %.1f%%)",
			d.Primary, primaryPercent, d.Secondary, secondaryPercent)
	}
}

// Validate validates the duo color data
func (d *DuoColorData) Validate() error {
	if d.Primary == "" {
		return fmt.Errorf("primary color cannot be empty")
	}
	if d.Secondary == "" {
		return fmt.Errorf("secondary color cannot be empty")
	}
	if d.Pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	if d.Ratio < 0.1 || d.Ratio > 0.9 {
		return fmt.Errorf("ratio must be between 0.1 and 0.9")
	}
	return nil
}

// RainbowColorData represents a rainbow color configuration
type RainbowColorData struct {
	Intensity   float64 `json:"intensity" validate:"min=0.1,max=1.0"`
	Saturation  float64 `json:"saturation" validate:"min=0.1,max=1.0"`
	Direction   string  `json:"direction" validate:"required"`
	Repetitions int     `json:"repetitions" validate:"min=1,max=10"`
}

// GetType returns the color type for rainbow color data
func (r *RainbowColorData) GetType() ColorType {
	return ColorTypeRainbow
}

// GenerateCSS generates CSS rainbow gradient representation
func (r *RainbowColorData) GenerateCSS() string {
	colors := []string{
		"#ff0000", "#ff8000", "#ffff00", "#80ff00",
		"#00ff00", "#00ff80", "#00ffff", "#0080ff",
		"#0000ff", "#8000ff", "#ff00ff", "#ff0080",
	}

	css := fmt.Sprintf("linear-gradient(%s", r.Direction)
	step := 100.0 / float64(len(colors))
	for i, color := range colors {
		position := float64(i) * step
		css += fmt.Sprintf(", %s %.1f%%", color, position)
	}
	css += ")"
	return css
}

// Validate validates the rainbow color data
func (r *RainbowColorData) Validate() error {
	if r.Direction == "" {
		return fmt.Errorf("direction cannot be empty")
	}
	if r.Intensity < 0.1 || r.Intensity > 1.0 {
		return fmt.Errorf("intensity must be between 0.1 and 1.0")
	}
	if r.Saturation < 0.1 || r.Saturation > 1.0 {
		return fmt.Errorf("saturation must be between 0.1 and 1.0")
	}
	if r.Repetitions < 1 || r.Repetitions > 10 {
		return fmt.Errorf("repetitions must be between 1 and 10")
	}
	return nil
}

// MetallicColorData represents a metallic color configuration
type MetallicColorData struct {
	BaseColor  string  `json:"base_color" validate:"required,hexcolor"`
	Shine      float64 `json:"shine" validate:"min=0.1,max=1.0"`
	Metallic   string  `json:"metallic_type" validate:"required"` // gold, silver, copper, bronze
	Highlights string  `json:"highlights,omitempty"`
}

// GetType returns the color type for metallic color data
func (m *MetallicColorData) GetType() ColorType {
	return ColorTypeMetallic
}

// GenerateCSS generates CSS for metallic colors
func (m *MetallicColorData) GenerateCSS() string {
	highlightColor := m.Highlights
	if highlightColor == "" {
		highlightColor = "#ffffff"
	}

	return fmt.Sprintf("linear-gradient(45deg, %s 0%%, %s 50%%, %s 100%%)",
		m.BaseColor, highlightColor, m.BaseColor)
}

// Validate validates the metallic color data
func (m *MetallicColorData) Validate() error {
	if m.BaseColor == "" {
		return fmt.Errorf("base color cannot be empty")
	}
	if m.Shine < 0.1 || m.Shine > 1.0 {
		return fmt.Errorf("shine must be between 0.1 and 1.0")
	}
	if m.Metallic == "" {
		return fmt.Errorf("metallic type cannot be empty")
	}
	return nil
}

// TransparentColorData represents a transparent/translucent color configuration
type TransparentColorData struct {
	BaseColor string  `json:"base_color" validate:"required,hexcolor"`
	Opacity   float64 `json:"opacity" validate:"min=0.1,max=0.9"`
	Clarity   string  `json:"clarity"` // clear, frosted, tinted
}

// GetType returns the color type for transparent color data
func (t *TransparentColorData) GetType() ColorType {
	return ColorTypeTransparent
}

// GenerateCSS generates CSS for transparent colors
func (t *TransparentColorData) GenerateCSS() string {
	return fmt.Sprintf("rgba(%s, %.2f)", t.BaseColor[1:], t.Opacity)
}

// Validate validates the transparent color data
func (t *TransparentColorData) Validate() error {
	if t.BaseColor == "" {
		return fmt.Errorf("base color cannot be empty")
	}
	if t.Opacity < 0.1 || t.Opacity > 0.9 {
		return fmt.Errorf("opacity must be between 0.1 and 0.9")
	}
	return nil
}

// ParseColorData parses JSON color data based on color type
func ParseColorData(colorType ColorType, dataJSON []byte) (ColorData, error) {
	switch colorType {
	case ColorTypeSolid:
		var data SolidColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse solid color data: %w", err)
		}
		return &data, data.Validate()

	case ColorTypeGradient:
		var data GradientColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse gradient color data: %w", err)
		}
		return &data, data.Validate()

	case ColorTypeDuo:
		var data DuoColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse duo color data: %w", err)
		}
		return &data, data.Validate()

	case ColorTypeRainbow:
		var data RainbowColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse rainbow color data: %w", err)
		}
		return &data, data.Validate()

	case ColorTypeMetallic:
		var data MetallicColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse metallic color data: %w", err)
		}
		return &data, data.Validate()

	case ColorTypeTransparent:
		var data TransparentColorData
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to parse transparent color data: %w", err)
		}
		return &data, data.Validate()

	default:
		return nil, fmt.Errorf("unsupported color type: %s", colorType)
	}
}

// MarshalColorData marshals color data to JSON
func MarshalColorData(data ColorData) ([]byte, error) {
	return json.Marshal(data)
}

// GenerateLegacyColorHex generates a hex color for backward compatibility
func GenerateLegacyColorHex(colorType ColorType, colorData ColorData) string {
	switch colorType {
	case ColorTypeSolid:
		if solid, ok := colorData.(*SolidColorData); ok {
			return solid.Color
		}
	case ColorTypeGradient:
		if gradient, ok := colorData.(*GradientColorData); ok && len(gradient.Colors) > 0 {
			return gradient.Colors[0].Color
		}
	case ColorTypeDuo:
		if duo, ok := colorData.(*DuoColorData); ok {
			return duo.Primary
		}
	case ColorTypeRainbow:
		return "#ff0000" // Red as default for rainbow
	case ColorTypeMetallic:
		if metallic, ok := colorData.(*MetallicColorData); ok {
			return metallic.BaseColor
		}
	case ColorTypeTransparent:
		if transparent, ok := colorData.(*TransparentColorData); ok {
			return transparent.BaseColor
		}
	}
	return "#000000" // Black as fallback
}
