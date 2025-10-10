package entities

import (
	"encoding/json"

	"github.com/google/uuid"
)

// CreateFilamentRequest represents the request to create a new filament
type CreateFilamentRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description,omitempty" validate:"max=1000"`
	BrandID     uuid.UUID `json:"brand_id" validate:"required"`
	MaterialID  uuid.UUID `json:"material_id" validate:"required"`

	// Color configuration
	Color     string          `json:"color" validate:"required,min=1,max=100"`
	ColorHex  string          `json:"color_hex,omitempty" validate:"omitempty,hexcolor"`
	ColorType ColorType       `json:"color_type,omitempty"`
	ColorData json.RawMessage `json:"color_data,omitempty"`

	// Physical properties
	Diameter   float64  `json:"diameter" validate:"required,gt=0,lte=10"`
	Weight     *float64 `json:"weight,omitempty" validate:"omitempty,gt=0"`
	PricePerKg float64  `json:"price_per_kg" validate:"required,gte=0"`

	// Additional properties
	URL string `json:"url,omitempty" validate:"omitempty,url"`

	// Technical specifications
	PrintTemperature *int `json:"print_temperature,omitempty" validate:"omitempty,gte=0,lte=500"`
	BedTemperature   *int `json:"bed_temperature,omitempty" validate:"omitempty,gte=0,lte=300"`

	// Ownership
	OwnerUserID *string `json:"owner_user_id,omitempty"`
}

// UpdateFilamentRequest represents the request to update an existing filament
type UpdateFilamentRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	BrandID     *uuid.UUID `json:"brand_id,omitempty"`
	MaterialID  *uuid.UUID `json:"material_id,omitempty"`

	// Color configuration
	Color     *string          `json:"color,omitempty" validate:"omitempty,min=1,max=100"`
	ColorHex  *string          `json:"color_hex,omitempty" validate:"omitempty,hexcolor"`
	ColorType *ColorType       `json:"color_type,omitempty"`
	ColorData *json.RawMessage `json:"color_data,omitempty"`

	// Physical properties
	Diameter   *float64 `json:"diameter,omitempty" validate:"omitempty,gt=0,lte=10"`
	Weight     *float64 `json:"weight,omitempty" validate:"omitempty,gt=0"`
	PricePerKg *float64 `json:"price_per_kg,omitempty" validate:"omitempty,gte=0"`

	// Additional properties
	URL *string `json:"url,omitempty" validate:"omitempty,url"`

	// Technical specifications
	PrintTemperature *int `json:"print_temperature,omitempty" validate:"omitempty,gte=0,lte=500"`
	BedTemperature   *int `json:"bed_temperature,omitempty" validate:"omitempty,gte=0,lte=300"`

	// Status
	IsActive *bool `json:"is_active,omitempty"`
}

// FilamentSearchRequest represents the request to search filaments
type FilamentSearchRequest struct {
	// Basic filters
	Name       *string    `json:"name,omitempty"`
	BrandID    *uuid.UUID `json:"brand_id,omitempty"`
	MaterialID *uuid.UUID `json:"material_id,omitempty"`
	Color      *string    `json:"color,omitempty"`
	ColorType  *ColorType `json:"color_type,omitempty"`

	// Price filters
	MinPrice *float64 `json:"min_price,omitempty"`
	MaxPrice *float64 `json:"max_price,omitempty"`

	// Diameter filters
	MinDiameter *float64 `json:"min_diameter,omitempty"`
	MaxDiameter *float64 `json:"max_diameter,omitempty"`

	// Technical filters
	MinPrintTemp *int  `json:"min_print_temp,omitempty"`
	MaxPrintTemp *int  `json:"max_print_temp,omitempty"`
	UVResistance *bool `json:"uv_resistance,omitempty"`

	// Ownership filters
	OnlyGlobal  *bool   `json:"only_global,omitempty"`
	OnlyUserOwn *bool   `json:"only_user_own,omitempty"`
	OwnerUserID *string `json:"owner_user_id,omitempty"`

	// Status filters
	IsActive *bool `json:"is_active,omitempty"`

	// Pagination
	Page     *int `json:"page,omitempty" validate:"omitempty,min=1"`
	PageSize *int `json:"page_size,omitempty" validate:"omitempty,min=1,max=100"`

	// Sorting
	SortBy    *string `json:"sort_by,omitempty"`    // name, price_per_kg, created_at, etc.
	SortOrder *string `json:"sort_order,omitempty"` // asc, desc
}
