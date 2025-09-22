package usecases

import (
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
}

// CreateFilamentRequest represents a request to create a new filament
type CreateFilamentRequest struct {
	Name          string   `json:"name" validate:"required,min=1,max=255"`
	Brand         string   `json:"brand" validate:"required,min=1,max=255"`
	Material      string   `json:"material" validate:"required,min=1,max=100"`
	Color         string   `json:"color" validate:"required,min=1,max=100"`
	ColorHex      string   `json:"color_hex,omitempty" validate:"omitempty,hexcolor"`
	Diameter      float64  `json:"diameter" validate:"required,min=0,max=10"`
	Weight        *float64 `json:"weight,omitempty" validate:"omitempty,min=0"`
	PricePerKg    float64  `json:"price_per_kg" validate:"required,min=0"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" validate:"omitempty,min=0"`
	URL           string   `json:"url,omitempty" validate:"omitempty,url"`
}

// UpdateFilamentRequest represents a request to update a filament
type UpdateFilamentRequest struct {
	Name          string   `json:"name" validate:"required,min=1,max=255"`
	Brand         string   `json:"brand" validate:"required,min=1,max=255"`
	Material      string   `json:"material" validate:"required,min=1,max=100"`
	Color         string   `json:"color" validate:"required,min=1,max=100"`
	ColorHex      string   `json:"color_hex,omitempty" validate:"omitempty,hexcolor"`
	Diameter      float64  `json:"diameter" validate:"required,min=0,max=10"`
	Weight        *float64 `json:"weight,omitempty" validate:"omitempty,min=0"`
	PricePerKg    float64  `json:"price_per_kg" validate:"required,min=0"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" validate:"omitempty,min=0"`
	URL           string   `json:"url,omitempty" validate:"omitempty,url"`
}

// FilamentResponse represents a filament in API responses
type FilamentResponse struct {
	ID            uint     `json:"id"`
	Name          string   `json:"name"`
	Brand         string   `json:"brand"`
	Material      string   `json:"material"`
	Color         string   `json:"color"`
	ColorHex      string   `json:"color_hex,omitempty"`
	Diameter      float64  `json:"diameter"`
	Weight        *float64 `json:"weight,omitempty"`
	PricePerKg    float64  `json:"price_per_kg"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty"`
	URL           string   `json:"url,omitempty"`
	OwnerUserID   *string  `json:"owner_user_id,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// ListResponse represents the response for filament lists
type ListResponse struct {
	Data []*FilamentResponse `json:"data"`
}

// ToFilamentResponse converts a filament entity to API response format
func ToFilamentResponse(filament *entities.Filament) *FilamentResponse {
	return &FilamentResponse{
		ID:            filament.ID,
		Name:          filament.Name,
		Brand:         filament.BrandName,
		Material:      filament.MaterialName,
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
}
