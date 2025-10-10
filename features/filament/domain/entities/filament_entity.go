package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FilamentEntity represents a 3D printing filament in the domain layer
type FilamentEntity struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`

	// Foreign keys for metadata
	BrandID    uuid.UUID `json:"brand_id"`
	MaterialID uuid.UUID `json:"material_id"`

	// Legacy color fields (maintained for backward compatibility)
	Color    string `json:"color"`
	ColorHex string `json:"color_hex,omitempty"`

	// Advanced color system
	ColorType    ColorType       `json:"color_type"`
	ColorData    json.RawMessage `json:"color_data,omitempty"`
	ColorPreview string          `json:"color_preview,omitempty"`

	// Physical properties
	Diameter   float64  `json:"diameter"`
	Weight     *float64 `json:"weight,omitempty"`
	PricePerKg float64  `json:"price_per_kg"`

	// Additional properties
	URL string `json:"url,omitempty"`

	// Ownership and access control
	OwnerUserID *string `json:"owner_user_id,omitempty"` // null = global catalog (admin), string = Keycloak User ID
	IsActive    bool    `json:"is_active"`

	// Technical specifications
	PrintTemperature *int `json:"print_temperature,omitempty"` // °C
	BedTemperature   *int `json:"bed_temperature,omitempty"`   // °C

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// IsGlobal verifies if the filament is from the global catalog (no owner)
func (f *FilamentEntity) IsGlobal() bool {
	return f.OwnerUserID == nil
}

// CanUserAccess verifies if a user can access this filament
func (f *FilamentEntity) CanUserAccess(userID string, isAdmin bool) bool {
	// Admin can access everything
	if isAdmin {
		return true
	}
	// Global filaments are accessible to all
	if f.IsGlobal() {
		return true
	}
	// User can access their own filaments
	return f.OwnerUserID != nil && *f.OwnerUserID == userID
}

// GetColorData parses and returns the structured color data
func (f *FilamentEntity) GetColorData() (ColorData, error) {
	if len(f.ColorData) == 0 || f.ColorType == "" {
		// Return legacy color data for backward compatibility
		return &SolidColorData{Color: f.ColorHex}, nil
	}

	return ParseColorData(f.ColorType, f.ColorData)
}

// SetColorData sets the color data from a ColorData interface
func (f *FilamentEntity) SetColorData(colorData ColorData) error {
	f.ColorType = colorData.GetType()

	dataBytes, err := MarshalColorData(colorData)
	if err != nil {
		return err
	}
	f.ColorData = json.RawMessage(dataBytes)

	f.ColorPreview = colorData.GenerateCSS()

	// Update legacy fields for backward compatibility
	f.ColorHex = GenerateLegacyColorHex(f.ColorType, colorData)

	return nil
}

// IsLegacyColor checks if this filament uses the legacy color system
func (f *FilamentEntity) IsLegacyColor() bool {
	return f.ColorType == "" || f.ColorType == ColorTypeSolid && len(f.ColorData) == 0
}

// MigrateToAdvancedColor migrates legacy color data to the new system
func (f *FilamentEntity) MigrateToAdvancedColor() error {
	if !f.IsLegacyColor() {
		return nil // Already using advanced system
	}

	// Create solid color data from legacy fields
	solidColor := &SolidColorData{
		Color: f.ColorHex,
	}

	if solidColor.Color == "" {
		solidColor.Color = "#000000" // Default to black if no color
	}

	return f.SetColorData(solidColor)
}

// Validate validates the filament entity
func (f *FilamentEntity) Validate() error {
	if f.Name == "" {
		return ErrFilamentNameRequired
	}
	if f.BrandID == uuid.Nil {
		return ErrFilamentBrandRequired
	}
	if f.MaterialID == uuid.Nil {
		return ErrFilamentMaterialRequired
	}
	if f.Diameter <= 0 {
		return ErrFilamentDiameterInvalid
	}
	if f.PricePerKg < 0 {
		return ErrFilamentPriceInvalid
	}

	// Validate color data if present
	if f.ColorType != "" && len(f.ColorData) > 0 {
		_, err := f.GetColorData()
		if err != nil {
			return err
		}
	}

	return nil
}
