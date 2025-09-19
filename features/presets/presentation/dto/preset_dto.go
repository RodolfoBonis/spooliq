package dto

import (
	"github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
)

// EnergyLocationResponse represents energy locations in API responses
type EnergyLocationResponse struct {
	Locations []string `json:"locations"`
}

// MachinePresetResponse represents machine preset in API responses
type MachinePresetResponse struct {
	Name        string  `json:"name"`
	Brand       string  `json:"brand"`
	Model       string  `json:"model"`
	Watt        float64 `json:"watt"`
	IdleFactor  float64 `json:"idle_factor"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
}

// EnergyPresetResponse represents energy preset in API responses
type EnergyPresetResponse struct {
	BaseTariff    float64 `json:"base_tariff"`
	FlagSurcharge float64 `json:"flag_surcharge"`
	Location      string  `json:"location"`
	Year          int     `json:"year"`
	Description   string  `json:"description,omitempty"`
}

// CreateEnergyPresetRequest represents request to create energy preset
type CreateEnergyPresetRequest struct {
	BaseTariff    float64 `json:"base_tariff" validate:"required,min=0"`
	FlagSurcharge float64 `json:"flag_surcharge" validate:"min=0"`
	Location      string  `json:"location" validate:"required"`
	Year          int     `json:"year" validate:"required,min=2020"`
	Description   string  `json:"description,omitempty"`
}

// CreateMachinePresetRequest represents request to create machine preset
type CreateMachinePresetRequest struct {
	Name        string  `json:"name" validate:"required"`
	Brand       string  `json:"brand" validate:"required"`
	Model       string  `json:"model" validate:"required"`
	Watt        float64 `json:"watt" validate:"required,min=0"`
	IdleFactor  float64 `json:"idle_factor" validate:"min=0,max=1"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
}

// UpdatePresetRequest represents request to update any preset
type UpdatePresetRequest struct {
	Data interface{} `json:"data" validate:"required"`
}

// MachinePresetsResponse represents machine presets list response
type MachinePresetsResponse struct {
	Machines []MachinePresetResponse `json:"machines"`
}

// EnergyPresetsResponse represents energy presets list response
type EnergyPresetsResponse struct {
	Presets []EnergyPresetResponse `json:"presets"`
}

// Conversion methods

// ToEntity converts CreateEnergyPresetRequest to domain entity
func (req *CreateEnergyPresetRequest) ToEntity() *entities.EnergyPreset {
	return &entities.EnergyPreset{
		BaseTariff:    req.BaseTariff,
		FlagSurcharge: req.FlagSurcharge,
		Location:      req.Location,
		Year:          req.Year,
		Description:   req.Description,
	}
}

// ToEntity converts CreateMachinePresetRequest to domain entity
func (req *CreateMachinePresetRequest) ToEntity() *entities.MachinePreset {
	return &entities.MachinePreset{
		Name:        req.Name,
		Brand:       req.Brand,
		Model:       req.Model,
		Watt:        req.Watt,
		IdleFactor:  req.IdleFactor,
		Description: req.Description,
		URL:         req.URL,
	}
}

// FromMachinePresetEntity converts domain entity to response DTO
func FromMachinePresetEntity(entity *entities.MachinePreset) MachinePresetResponse {
	return MachinePresetResponse{
		Name:        entity.Name,
		Brand:       entity.Brand,
		Model:       entity.Model,
		Watt:        entity.Watt,
		IdleFactor:  entity.IdleFactor,
		Description: entity.Description,
		URL:         entity.URL,
	}
}

// FromEnergyPresetEntity converts domain entity to response DTO
func FromEnergyPresetEntity(entity *entities.EnergyPreset) EnergyPresetResponse {
	return EnergyPresetResponse{
		BaseTariff:    entity.BaseTariff,
		FlagSurcharge: entity.FlagSurcharge,
		Location:      entity.Location,
		Year:          entity.Year,
		Description:   entity.Description,
	}
}

// FromMachinePresetEntities converts slice of domain entities to response DTOs
func FromMachinePresetEntities(entities []*entities.MachinePreset) []MachinePresetResponse {
	responses := make([]MachinePresetResponse, len(entities))
	for i, entity := range entities {
		responses[i] = FromMachinePresetEntity(entity)
	}
	return responses
}

// FromEnergyPresetEntities converts slice of domain entities to response DTOs
func FromEnergyPresetEntities(entities []*entities.EnergyPreset) []EnergyPresetResponse {
	responses := make([]EnergyPresetResponse, len(entities))
	for i, entity := range entities {
		responses[i] = FromEnergyPresetEntity(entity)
	}
	return responses
}
