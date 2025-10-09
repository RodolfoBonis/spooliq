package usecases

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// CreatePresetUseCase handles creating new presets
type CreatePresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewCreatePresetUseCase creates a new instance of CreatePresetUseCase
func NewCreatePresetUseCase(presetRepo repositories.PresetRepository) *CreatePresetUseCase {
	return &CreatePresetUseCase{
		presetRepo: presetRepo,
	}
}

// CreateMachinePresetRequest represents the request to create a machine preset
type CreateMachinePresetRequest struct {
	Name                   string     `json:"name" binding:"required"`
	Description            string     `json:"description"`
	IsDefault              bool       `json:"is_default"`
	UserID                 *uuid.UUID `json:"user_id"`
	Brand                  string     `json:"brand"`
	Model                  string     `json:"model"`
	BuildVolumeX           float32    `json:"build_volume_x" binding:"required,gt=0"`
	BuildVolumeY           float32    `json:"build_volume_y" binding:"required,gt=0"`
	BuildVolumeZ           float32    `json:"build_volume_z" binding:"required,gt=0"`
	NozzleDiameter         float32    `json:"nozzle_diameter" binding:"required,gt=0"`
	LayerHeightMin         float32    `json:"layer_height_min" binding:"required,gt=0"`
	LayerHeightMax         float32    `json:"layer_height_max" binding:"required,gt=0"`
	PrintSpeedMax          float32    `json:"print_speed_max" binding:"required,gt=0"`
	PowerConsumption       float32    `json:"power_consumption" binding:"required,gt=0"`
	BedTemperatureMax      float32    `json:"bed_temperature_max"`
	ExtruderTemperatureMax float32    `json:"extruder_temperature_max"`
	FilamentDiameter       float32    `json:"filament_diameter" binding:"required,gt=0"`
	CostPerHour            float32    `json:"cost_per_hour"`
}

// CreateEnergyPresetRequest represents the request to create an energy preset
type CreateEnergyPresetRequest struct {
	Name                  string     `json:"name" binding:"required"`
	Description           string     `json:"description"`
	IsDefault             bool       `json:"is_default"`
	UserID                *uuid.UUID `json:"user_id"`
	Country               string     `json:"country"`
	State                 string     `json:"state"`
	City                  string     `json:"city"`
	EnergyCostPerKwh      float32    `json:"energy_cost_per_kwh" binding:"required,gt=0"`
	Currency              string     `json:"currency" binding:"required,len=3"`
	Provider              string     `json:"provider"`
	TariffType            string     `json:"tariff_type"`
	PeakHourMultiplier    float32    `json:"peak_hour_multiplier" binding:"required,gt=0"`
	OffPeakHourMultiplier float32    `json:"off_peak_hour_multiplier" binding:"required,gt=0"`
}

// CreateCostPresetRequest represents the request to create a cost preset
type CreateCostPresetRequest struct {
	Name                      string     `json:"name" binding:"required"`
	Description               string     `json:"description"`
	IsDefault                 bool       `json:"is_default"`
	UserID                    *uuid.UUID `json:"user_id"`
	LaborCostPerHour          float32    `json:"labor_cost_per_hour" binding:"min=0"`
	PackagingCostPerItem      float32    `json:"packaging_cost_per_item" binding:"min=0"`
	ShippingCostBase          float32    `json:"shipping_cost_base" binding:"min=0"`
	ShippingCostPerGram       float32    `json:"shipping_cost_per_gram" binding:"min=0"`
	OverheadPercentage        float32    `json:"overhead_percentage" binding:"min=0,max=100"`
	ProfitMarginPercentage    float32    `json:"profit_margin_percentage" binding:"min=0,max=100"`
	PostProcessingCostPerHour float32    `json:"post_processing_cost_per_hour" binding:"min=0"`
	SupportRemovalCostPerHour float32    `json:"support_removal_cost_per_hour" binding:"min=0"`
	QualityControlCostPerItem float32    `json:"quality_control_cost_per_item" binding:"min=0"`
}

// CreateMachinePreset Execute creates a new machine preset
func (uc *CreatePresetUseCase) CreateMachinePreset(req *CreateMachinePresetRequest) (*entities.PresetEntity, error) {
	// Create base preset entity
	preset := &entities.PresetEntity{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Type:        entities.PresetTypeMachine,
		IsActive:    true,
		IsDefault:   req.IsDefault,
		UserID:      req.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate base preset
	if err := preset.Validate(); err != nil {
		return nil, err
	}

	// Create machine-specific entity
	machine := &entities.MachinePresetEntity{
		ID:                     preset.ID,
		Brand:                  req.Brand,
		Model:                  req.Model,
		BuildVolumeX:           req.BuildVolumeX,
		BuildVolumeY:           req.BuildVolumeY,
		BuildVolumeZ:           req.BuildVolumeZ,
		NozzleDiameter:         req.NozzleDiameter,
		LayerHeightMin:         req.LayerHeightMin,
		LayerHeightMax:         req.LayerHeightMax,
		PrintSpeedMax:          req.PrintSpeedMax,
		PowerConsumption:       req.PowerConsumption,
		BedTemperatureMax:      req.BedTemperatureMax,
		ExtruderTemperatureMax: req.ExtruderTemperatureMax,
		FilamentDiameter:       req.FilamentDiameter,
		CostPerHour:            req.CostPerHour,
	}

	// Validate machine preset
	if err := machine.Validate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := uc.presetRepo.CreateMachine(preset, machine); err != nil {
		return nil, err
	}

	return preset, nil
}

// CreateEnergyPreset creates a new energy preset
func (uc *CreatePresetUseCase) CreateEnergyPreset(req *CreateEnergyPresetRequest) (*entities.PresetEntity, error) {
	// Create base preset entity
	preset := &entities.PresetEntity{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Type:        entities.PresetTypeEnergy,
		IsActive:    true,
		IsDefault:   req.IsDefault,
		UserID:      req.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate base preset
	if err := preset.Validate(); err != nil {
		return nil, err
	}

	// Create energy-specific entity
	energy := &entities.EnergyPresetEntity{
		ID:                    preset.ID,
		Country:               req.Country,
		State:                 req.State,
		City:                  req.City,
		EnergyCostPerKwh:      req.EnergyCostPerKwh,
		Currency:              req.Currency,
		Provider:              req.Provider,
		TariffType:            req.TariffType,
		PeakHourMultiplier:    req.PeakHourMultiplier,
		OffPeakHourMultiplier: req.OffPeakHourMultiplier,
	}

	// Validate energy preset
	if err := energy.Validate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := uc.presetRepo.CreateEnergy(preset, energy); err != nil {
		return nil, err
	}

	return preset, nil
}

// CreateCostPreset creates a new cost preset
func (uc *CreatePresetUseCase) CreateCostPreset(req *CreateCostPresetRequest) (*entities.PresetEntity, error) {
	// Create base preset entity
	preset := &entities.PresetEntity{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Type:        entities.PresetTypeCost,
		IsActive:    true,
		IsDefault:   req.IsDefault,
		UserID:      req.UserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Validate base preset
	if err := preset.Validate(); err != nil {
		return nil, err
	}

	// Create cost-specific entity
	cost := &entities.CostPresetEntity{
		ID:                        preset.ID,
		LaborCostPerHour:          req.LaborCostPerHour,
		PackagingCostPerItem:      req.PackagingCostPerItem,
		ShippingCostBase:          req.ShippingCostBase,
		ShippingCostPerGram:       req.ShippingCostPerGram,
		OverheadPercentage:        req.OverheadPercentage,
		ProfitMarginPercentage:    req.ProfitMarginPercentage,
		PostProcessingCostPerHour: req.PostProcessingCostPerHour,
		SupportRemovalCostPerHour: req.SupportRemovalCostPerHour,
		QualityControlCostPerItem: req.QualityControlCostPerItem,
	}

	// Validate cost preset
	if err := cost.Validate(); err != nil {
		return nil, err
	}

	// Save to repository
	if err := uc.presetRepo.CreateCost(preset, cost); err != nil {
		return nil, err
	}

	return preset, nil
}
