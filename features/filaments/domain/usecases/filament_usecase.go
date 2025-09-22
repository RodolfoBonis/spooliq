package usecases

import (
	"encoding/json"
	"fmt"

	"github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/gin-gonic/gin"
)

// FilamentUseCase defines operations for filament management
type FilamentUseCase interface {
	CreateFilament(c *gin.Context)
	GetFilament(c *gin.Context)
	GetAllFilaments(c *gin.Context)
	UpdateFilament(c *gin.Context)
	DeleteFilament(c *gin.Context)
	GetUserFilaments(c *gin.Context)
	GetGlobalFilaments(c *gin.Context)
	MigrateFilamentToAdvancedColor(c *gin.Context)
}

// CreateFilamentRequest represents a request to create a new filament
// @Description Request body for creating a new filament with support for both legacy and advanced color systems
type CreateFilamentRequest struct {
	Name       string `json:"name" validate:"required,min=1,max=255" example:"PLA Premium White"`
	BrandID    uint   `json:"brand_id" validate:"required" example:"1"`
	MaterialID uint   `json:"material_id" validate:"required" example:"1"`

	// Legacy color fields (maintained for backward compatibility)
	Color    string `json:"color" validate:"required,min=1,max=100" example:"White"`
	ColorHex string `json:"color_hex,omitempty" validate:"omitempty,hexcolor" example:"#FFFFFF"`

	// Advanced color system (optional, takes precedence over legacy fields if provided)
	ColorType entities.ColorType `json:"color_type,omitempty" validate:"omitempty" example:"solid" enums:"solid,gradient,duo,rainbow"`
	ColorData json.RawMessage    `json:"color_data,omitempty" swaggertype:"object,string"`

	Diameter      float64  `json:"diameter" validate:"required,min=0,max=10" example:"1.75"`
	Weight        *float64 `json:"weight,omitempty" validate:"omitempty,min=0" example:"1000"`
	PricePerKg    float64  `json:"price_per_kg" validate:"required,min=0" example:"125.50"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" validate:"omitempty,min=0" example:"0.05"`
	URL           string   `json:"url,omitempty" validate:"omitempty,url" example:"https://example.com/filament"`
}

// UpdateFilamentRequest represents a request to update a filament
// @Description Request body for updating an existing filament with support for both legacy and advanced color systems
type UpdateFilamentRequest struct {
	Name       string `json:"name" validate:"required,min=1,max=255" example:"PLA Premium White"`
	BrandID    uint   `json:"brand_id" validate:"required" example:"1"`
	MaterialID uint   `json:"material_id" validate:"required" example:"1"`

	// Legacy color fields (maintained for backward compatibility)
	Color    string `json:"color" validate:"required,min=1,max=100" example:"White"`
	ColorHex string `json:"color_hex,omitempty" validate:"omitempty,hexcolor" example:"#FFFFFF"`

	// Advanced color system (optional, takes precedence over legacy fields if provided)
	ColorType entities.ColorType `json:"color_type,omitempty" validate:"omitempty" example:"gradient" enums:"solid,gradient,duo,rainbow"`
	ColorData json.RawMessage    `json:"color_data,omitempty" swaggertype:"object,string"`

	Diameter      float64  `json:"diameter" validate:"required,min=0,max=10" example:"1.75"`
	Weight        *float64 `json:"weight,omitempty" validate:"omitempty,min=0" example:"1000"`
	PricePerKg    float64  `json:"price_per_kg" validate:"required,min=0" example:"125.50"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" validate:"omitempty,min=0" example:"0.05"`
	URL           string   `json:"url,omitempty" validate:"omitempty,url" example:"https://example.com/filament"`
}

// FilamentResponse represents a filament in API responses
// @Description Response object for filament data with support for both legacy and advanced color systems
type FilamentResponse struct {
	ID       uint   `json:"id" example:"1"`
	Name     string `json:"name" example:"PLA Premium White"`
	Brand    string `json:"brand" example:"SUNLU"`
	Material string `json:"material" example:"PLA"`

	// Legacy color fields (always provided for backward compatibility)
	Color    string `json:"color" example:"White"`
	ColorHex string `json:"color_hex,omitempty" example:"#FFFFFF"`

	// Advanced color system (provided when available)
	ColorType    entities.ColorType `json:"color_type,omitempty" example:"solid" enums:"solid,gradient,duo,rainbow"`
	ColorData    json.RawMessage    `json:"color_data,omitempty" swaggertype:"object,string"`
	ColorPreview string             `json:"color_preview,omitempty" example:"#FFFFFF"`

	Diameter      float64  `json:"diameter" example:"1.75"`
	Weight        *float64 `json:"weight,omitempty" example:"1000"`
	PricePerKg    float64  `json:"price_per_kg" example:"125.50"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" example:"0.05"`
	URL           string   `json:"url,omitempty" example:"https://example.com/filament"`
	OwnerUserID   *string  `json:"owner_user_id,omitempty" example:"user-uuid-123"`
	CreatedAt     string   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt     string   `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ListResponse represents the response for filament lists
// @Description Response object containing a list of filaments
type ListResponse struct {
	Data []*FilamentResponse `json:"data"`
}

// ToFilamentResponse converts a filament entity to API response format
func ToFilamentResponse(filament *entities.Filament) *FilamentResponse {
	response := &FilamentResponse{
		ID:            filament.ID,
		Name:          filament.Name,
		Brand:         filament.Brand.Name,
		Material:      filament.Material.Name,
		Color:         filament.Color,
		ColorHex:      filament.ColorHex,
		Diameter:      filament.Diameter,
		Weight:        filament.Weight,
		PricePerKg:    filament.PricePerKg,
		PricePerMeter: filament.PricePerMeter,
		URL:           filament.URL,
		OwnerUserID:   filament.OwnerUserID,
		CreatedAt:     filament.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     filament.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Include advanced color data if available
	if filament.ColorType != "" {
		response.ColorType = filament.ColorType
		if filament.ColorData != "" {
			response.ColorData = json.RawMessage(filament.ColorData)
		}
		response.ColorPreview = filament.ColorPreview
	}

	return response
}

// IsUsingAdvancedColor checks if the request contains advanced color data
func (r *CreateFilamentRequest) IsUsingAdvancedColor() bool {
	return r.ColorType != "" && len(r.ColorData) > 0
}

// ValidateColorData validates the color data based on the color type
func (r *CreateFilamentRequest) ValidateColorData() error {
	if !r.IsUsingAdvancedColor() {
		return nil // No validation needed for legacy color system
	}

	if !r.ColorType.IsValid() {
		return fmt.Errorf("invalid color_type: %s", r.ColorType)
	}

	_, err := entities.ParseColorData(r.ColorType, r.ColorData)
	return err
}

// GetColorData returns the parsed color data from the request
func (r *CreateFilamentRequest) GetColorData() (entities.ColorData, error) {
	if !r.IsUsingAdvancedColor() {
		// Return legacy color data
		return &entities.SolidColorData{Color: r.ColorHex}, nil
	}

	return entities.ParseColorData(r.ColorType, r.ColorData)
}

// IsUsingAdvancedColor checks if the request contains advanced color data
func (r *UpdateFilamentRequest) IsUsingAdvancedColor() bool {
	return r.ColorType != "" && len(r.ColorData) > 0
}

// ValidateColorData validates the color data based on the color type
func (r *UpdateFilamentRequest) ValidateColorData() error {
	if !r.IsUsingAdvancedColor() {
		return nil // No validation needed for legacy color system
	}

	if !r.ColorType.IsValid() {
		return fmt.Errorf("invalid color_type: %s", r.ColorType)
	}

	_, err := entities.ParseColorData(r.ColorType, r.ColorData)
	return err
}

// GetColorData returns the parsed color data from the request
func (r *UpdateFilamentRequest) GetColorData() (entities.ColorData, error) {
	if !r.IsUsingAdvancedColor() {
		// Return legacy color data
		return &entities.SolidColorData{Color: r.ColorHex}, nil
	}

	return entities.ParseColorData(r.ColorType, r.ColorData)
}
