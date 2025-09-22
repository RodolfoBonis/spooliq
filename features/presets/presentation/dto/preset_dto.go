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
	Key            string            `json:"key,omitempty"`
	Name           string            `json:"name"`
	Brand          string            `json:"brand"`
	Model          string            `json:"model"`
	Watt           float64           `json:"watt"`
	IdleFactor     float64           `json:"idle_factor"`
	Description    string            `json:"description,omitempty"`
	URL            string            `json:"url,omitempty"`
	BuildVolume    *BuildVolumeDTO   `json:"build_volume,omitempty"`
	NozzleDiameter float64           `json:"nozzle_diameter,omitempty"`
	MaxTemperature int               `json:"max_temperature,omitempty"`
	HeatedBed      bool              `json:"heated_bed,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
	UpdatedAt      string            `json:"updated_at,omitempty"`
}

// BuildVolumeDTO represents build volume in DTOs
type BuildVolumeDTO struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// EnergyPresetResponse represents energy preset in API responses
type EnergyPresetResponse struct {
	Key           string `json:"key,omitempty"`
	BaseTariff    float64 `json:"base_tariff"`
	FlagSurcharge float64 `json:"flag_surcharge"`
	Location      string  `json:"location"`
	State         string  `json:"state,omitempty"`
	City          string  `json:"city,omitempty"`
	Year          int     `json:"year"`
	Month         *int    `json:"month,omitempty"`
	FlagType      string  `json:"flag_type,omitempty"`
	Description   string  `json:"description,omitempty"`
	CreatedAt     string  `json:"created_at,omitempty"`
	UpdatedAt     string  `json:"updated_at,omitempty"`
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

// CostPresetResponse represents cost preset in API responses
type CostPresetResponse struct {
	Key             string  `json:"key,omitempty"`
	Name            string  `json:"name"`
	Description     string  `json:"description,omitempty"`
	OverheadAmount  float64 `json:"overhead_amount"`
	WearPercentage  float64 `json:"wear_percentage"`
	IsDefault       bool    `json:"is_default,omitempty"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

// MarginPresetResponse represents margin preset in API responses
type MarginPresetResponse struct {
	Key                   string  `json:"key,omitempty"`
	Name                  string  `json:"name"`
	Description           string  `json:"description,omitempty"`
	PrintingOnlyMargin    float64 `json:"printing_only_margin"`
	PrintingPlusMargin    float64 `json:"printing_plus_margin"`
	FullServiceMargin     float64 `json:"full_service_margin"`
	OperatorRatePerHour   float64 `json:"operator_rate_per_hour"`
	ModelerRatePerHour    float64 `json:"modeler_rate_per_hour"`
	IsDefault             bool    `json:"is_default,omitempty"`
	CreatedAt             string  `json:"created_at,omitempty"`
	UpdatedAt             string  `json:"updated_at,omitempty"`
}

// CostPresetsResponse represents cost presets list response
type CostPresetsResponse struct {
	CostPresets []CostPresetResponse `json:"cost_presets"`
}

// MarginPresetsResponse represents margin presets list response
type MarginPresetsResponse struct {
	MarginPresets []MarginPresetResponse `json:"margin_presets"`
}

// CreateCostPresetRequest represents request to create cost preset
type CreateCostPresetRequest struct {
	Name            string  `json:"name" validate:"required,min=1,max=100"`
	Description     string  `json:"description,omitempty"`
	OverheadAmount  float64 `json:"overhead_amount" validate:"required,min=0"`
	WearPercentage  float64 `json:"wear_percentage" validate:"required,min=0,max=100"`
	IsDefault       bool    `json:"is_default,omitempty"`
}

// CreateMarginPresetRequest represents request to create margin preset
type CreateMarginPresetRequest struct {
	Name                  string  `json:"name" validate:"required,min=1,max=100"`
	Description           string  `json:"description,omitempty"`
	PrintingOnlyMargin    float64 `json:"printing_only_margin" validate:"required,min=0"`
	PrintingPlusMargin    float64 `json:"printing_plus_margin" validate:"required,min=0"`
	FullServiceMargin     float64 `json:"full_service_margin" validate:"required,min=0"`
	OperatorRatePerHour   float64 `json:"operator_rate_per_hour" validate:"required,min=0"`
	ModelerRatePerHour    float64 `json:"modeler_rate_per_hour" validate:"required,min=0"`
	IsDefault             bool    `json:"is_default,omitempty"`
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
	var buildVolume *BuildVolumeDTO
	if entity.BuildVolume != nil {
		buildVolume = &BuildVolumeDTO{
			X: entity.BuildVolume.X,
			Y: entity.BuildVolume.Y,
			Z: entity.BuildVolume.Z,
		}
	}

	return MachinePresetResponse{
		Key:            entity.Key,
		Name:           entity.Name,
		Brand:          entity.Brand,
		Model:          entity.Model,
		Watt:           entity.Watt,
		IdleFactor:     entity.IdleFactor,
		Description:    entity.Description,
		URL:            entity.URL,
		BuildVolume:    buildVolume,
		NozzleDiameter: entity.NozzleDiameter,
		MaxTemperature: entity.MaxTemperature,
		HeatedBed:      entity.HeatedBed,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}

// FromEnergyPresetEntity converts domain entity to response DTO
func FromEnergyPresetEntity(entity *entities.EnergyPreset) EnergyPresetResponse {
	return EnergyPresetResponse{
		Key:           entity.Key,
		BaseTariff:    entity.BaseTariff,
		FlagSurcharge: entity.FlagSurcharge,
		Location:      entity.Location,
		State:         entity.State,
		City:          entity.City,
		Year:          entity.Year,
		Month:         entity.Month,
		FlagType:      entity.FlagType,
		Description:   entity.Description,
		CreatedAt:     entity.CreatedAt,
		UpdatedAt:     entity.UpdatedAt,
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

// ToEntity converts CreateCostPresetRequest to domain entity
func (req *CreateCostPresetRequest) ToEntity() *entities.CostPreset {
	return &entities.CostPreset{
		Name:            req.Name,
		Description:     req.Description,
		OverheadAmount:  req.OverheadAmount,
		WearPercentage:  req.WearPercentage,
		IsDefault:       req.IsDefault,
	}
}

// ToEntity converts CreateMarginPresetRequest to domain entity
func (req *CreateMarginPresetRequest) ToEntity() *entities.MarginPreset {
	return &entities.MarginPreset{
		Name:                  req.Name,
		Description:           req.Description,
		PrintingOnlyMargin:    req.PrintingOnlyMargin,
		PrintingPlusMargin:    req.PrintingPlusMargin,
		FullServiceMargin:     req.FullServiceMargin,
		OperatorRatePerHour:   req.OperatorRatePerHour,
		ModelerRatePerHour:    req.ModelerRatePerHour,
		IsDefault:             req.IsDefault,
	}
}

// FromCostPresetEntity converts domain entity to response DTO
func FromCostPresetEntity(entity *entities.CostPreset) CostPresetResponse {
	return CostPresetResponse{
		Key:             entity.Key,
		Name:            entity.Name,
		Description:     entity.Description,
		OverheadAmount:  entity.OverheadAmount,
		WearPercentage:  entity.WearPercentage,
		IsDefault:       entity.IsDefault,
		CreatedAt:       entity.CreatedAt,
		UpdatedAt:       entity.UpdatedAt,
	}
}

// FromMarginPresetEntity converts domain entity to response DTO
func FromMarginPresetEntity(entity *entities.MarginPreset) MarginPresetResponse {
	return MarginPresetResponse{
		Key:                   entity.Key,
		Name:                  entity.Name,
		Description:           entity.Description,
		PrintingOnlyMargin:    entity.PrintingOnlyMargin,
		PrintingPlusMargin:    entity.PrintingPlusMargin,
		FullServiceMargin:     entity.FullServiceMargin,
		OperatorRatePerHour:   entity.OperatorRatePerHour,
		ModelerRatePerHour:    entity.ModelerRatePerHour,
		IsDefault:             entity.IsDefault,
		CreatedAt:             entity.CreatedAt,
		UpdatedAt:             entity.UpdatedAt,
	}
}

// FromCostPresetEntities converts slice of domain entities to response DTOs
func FromCostPresetEntities(entities []*entities.CostPreset) []CostPresetResponse {
	responses := make([]CostPresetResponse, len(entities))
	for i, entity := range entities {
		responses[i] = FromCostPresetEntity(entity)
	}
	return responses
}

// FromMarginPresetEntities converts slice of domain entities to response DTOs
func FromMarginPresetEntities(entities []*entities.MarginPreset) []MarginPresetResponse {
	responses := make([]MarginPresetResponse, len(entities))
	for i, entity := range entities {
		responses[i] = FromMarginPresetEntity(entity)
	}
	return responses
}
