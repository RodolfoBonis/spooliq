package usecases

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// UpdatePresetUseCase handles updating existing presets
type UpdatePresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewUpdatePresetUseCase creates a new instance of UpdatePresetUseCase
func NewUpdatePresetUseCase(presetRepo repositories.PresetRepository) *UpdatePresetUseCase {
	return &UpdatePresetUseCase{
		presetRepo: presetRepo,
	}
}

// UpdateMachinePresetRequest represents the request to update a machine preset
type UpdateMachinePresetRequest struct {
	ID                     uuid.UUID `json:"id" binding:"required"`
	Name                   string    `json:"name"`
	Description            string    `json:"description"`
	IsActive               *bool     `json:"is_active"`
	IsDefault              *bool     `json:"is_default"`
	Brand                  string    `json:"brand"`
	Model                  string    `json:"model"`
	BuildVolumeX           *float32  `json:"build_volume_x"`
	BuildVolumeY           *float32  `json:"build_volume_y"`
	BuildVolumeZ           *float32  `json:"build_volume_z"`
	NozzleDiameter         *float32  `json:"nozzle_diameter"`
	LayerHeightMin         *float32  `json:"layer_height_min"`
	LayerHeightMax         *float32  `json:"layer_height_max"`
	PrintSpeedMax          *float32  `json:"print_speed_max"`
	PowerConsumption       *float32  `json:"power_consumption"`
	BedTemperatureMax      *float32  `json:"bed_temperature_max"`
	ExtruderTemperatureMax *float32  `json:"extruder_temperature_max"`
	FilamentDiameter       *float32  `json:"filament_diameter"`
	CostPerHour            *float32  `json:"cost_per_hour"`
}

// UpdateEnergyPresetRequest represents the request to update an energy preset
type UpdateEnergyPresetRequest struct {
	ID                    uuid.UUID `json:"id" binding:"required"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	IsActive              *bool     `json:"is_active"`
	IsDefault             *bool     `json:"is_default"`
	Country               string    `json:"country"`
	State                 string    `json:"state"`
	City                  string    `json:"city"`
	EnergyCostPerKwh      *float32  `json:"energy_cost_per_kwh"`
	Currency              string    `json:"currency"`
	Provider              string    `json:"provider"`
	TariffType            string    `json:"tariff_type"`
	PeakHourMultiplier    *float32  `json:"peak_hour_multiplier"`
	OffPeakHourMultiplier *float32  `json:"off_peak_hour_multiplier"`
}

// UpdateCostPresetRequest represents the request to update a cost preset
type UpdateCostPresetRequest struct {
	ID                        uuid.UUID `json:"id" binding:"required"`
	Name                      string    `json:"name"`
	Description               string    `json:"description"`
	IsActive                  *bool     `json:"is_active"`
	IsDefault                 *bool     `json:"is_default"`
	LaborCostPerHour          *float32  `json:"labor_cost_per_hour"`
	PackagingCostPerItem      *float32  `json:"packaging_cost_per_item"`
	ShippingCostBase          *float32  `json:"shipping_cost_base"`
	ShippingCostPerGram       *float32  `json:"shipping_cost_per_gram"`
	OverheadPercentage        *float32  `json:"overhead_percentage"`
	ProfitMarginPercentage    *float32  `json:"profit_margin_percentage"`
	PostProcessingCostPerHour *float32  `json:"post_processing_cost_per_hour"`
	SupportRemovalCostPerHour *float32  `json:"support_removal_cost_per_hour"`
	QualityControlCostPerItem *float32  `json:"quality_control_cost_per_item"`
}

// UpdateMachinePreset updates an existing machine preset
func (uc *UpdatePresetUseCase) UpdateMachinePreset(req *UpdateMachinePresetRequest) (*entities.PresetEntity, error) {
	// Get existing preset
	preset, err := uc.presetRepo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Verify it's a machine preset
	if preset.Type != entities.PresetTypeMachine {
		return nil, entities.ErrInvalidPresetType
	}

	// Get existing machine data
	machine, err := uc.presetRepo.GetMachineByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Update base preset fields
	if req.Name != "" {
		preset.Name = req.Name
	}
	if req.Description != "" {
		preset.Description = req.Description
	}
	if req.IsActive != nil {
		preset.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		preset.IsDefault = *req.IsDefault
	}
	preset.UpdatedAt = time.Now()

	// Update machine-specific fields
	if req.Brand != "" {
		machine.Brand = req.Brand
	}
	if req.Model != "" {
		machine.Model = req.Model
	}
	if req.BuildVolumeX != nil {
		machine.BuildVolumeX = *req.BuildVolumeX
	}
	if req.BuildVolumeY != nil {
		machine.BuildVolumeY = *req.BuildVolumeY
	}
	if req.BuildVolumeZ != nil {
		machine.BuildVolumeZ = *req.BuildVolumeZ
	}
	if req.NozzleDiameter != nil {
		machine.NozzleDiameter = *req.NozzleDiameter
	}
	if req.LayerHeightMin != nil {
		machine.LayerHeightMin = *req.LayerHeightMin
	}
	if req.LayerHeightMax != nil {
		machine.LayerHeightMax = *req.LayerHeightMax
	}
	if req.PrintSpeedMax != nil {
		machine.PrintSpeedMax = *req.PrintSpeedMax
	}
	if req.PowerConsumption != nil {
		machine.PowerConsumption = *req.PowerConsumption
	}
	if req.BedTemperatureMax != nil {
		machine.BedTemperatureMax = *req.BedTemperatureMax
	}
	if req.ExtruderTemperatureMax != nil {
		machine.ExtruderTemperatureMax = *req.ExtruderTemperatureMax
	}
	if req.FilamentDiameter != nil {
		machine.FilamentDiameter = *req.FilamentDiameter
	}
	if req.CostPerHour != nil {
		machine.CostPerHour = *req.CostPerHour
	}

	// Validate updated entities
	if err := preset.Validate(); err != nil {
		return nil, err
	}
	if err := machine.Validate(); err != nil {
		return nil, err
	}

	// Save updates
	if err := uc.presetRepo.Update(preset); err != nil {
		return nil, err
	}
	if err := uc.presetRepo.UpdateMachine(machine); err != nil {
		return nil, err
	}

	return preset, nil
}

// UpdateEnergyPreset updates an existing energy preset
func (uc *UpdatePresetUseCase) UpdateEnergyPreset(req *UpdateEnergyPresetRequest) (*entities.PresetEntity, error) {
	// Get existing preset
	preset, err := uc.presetRepo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Verify it's an energy preset
	if preset.Type != entities.PresetTypeEnergy {
		return nil, entities.ErrInvalidPresetType
	}

	// Get existing energy data
	energy, err := uc.presetRepo.GetEnergyByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Update base preset fields
	if req.Name != "" {
		preset.Name = req.Name
	}
	if req.Description != "" {
		preset.Description = req.Description
	}
	if req.IsActive != nil {
		preset.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		preset.IsDefault = *req.IsDefault
	}
	preset.UpdatedAt = time.Now()

	// Update energy-specific fields
	if req.Country != "" {
		energy.Country = req.Country
	}
	if req.State != "" {
		energy.State = req.State
	}
	if req.City != "" {
		energy.City = req.City
	}
	if req.EnergyCostPerKwh != nil {
		energy.EnergyCostPerKwh = *req.EnergyCostPerKwh
	}
	if req.Currency != "" {
		energy.Currency = req.Currency
	}
	if req.Provider != "" {
		energy.Provider = req.Provider
	}
	if req.TariffType != "" {
		energy.TariffType = req.TariffType
	}
	if req.PeakHourMultiplier != nil {
		energy.PeakHourMultiplier = *req.PeakHourMultiplier
	}
	if req.OffPeakHourMultiplier != nil {
		energy.OffPeakHourMultiplier = *req.OffPeakHourMultiplier
	}

	// Validate updated entities
	if err := preset.Validate(); err != nil {
		return nil, err
	}
	if err := energy.Validate(); err != nil {
		return nil, err
	}

	// Save updates
	if err := uc.presetRepo.Update(preset); err != nil {
		return nil, err
	}
	if err := uc.presetRepo.UpdateEnergy(energy); err != nil {
		return nil, err
	}

	return preset, nil
}

// UpdateCostPreset updates an existing cost preset
func (uc *UpdatePresetUseCase) UpdateCostPreset(req *UpdateCostPresetRequest) (*entities.PresetEntity, error) {
	// Get existing preset
	preset, err := uc.presetRepo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Verify it's a cost preset
	if preset.Type != entities.PresetTypeCost {
		return nil, entities.ErrInvalidPresetType
	}

	// Get existing cost data
	cost, err := uc.presetRepo.GetCostByID(req.ID)
	if err != nil {
		return nil, err
	}

	// Update base preset fields
	if req.Name != "" {
		preset.Name = req.Name
	}
	if req.Description != "" {
		preset.Description = req.Description
	}
	if req.IsActive != nil {
		preset.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		preset.IsDefault = *req.IsDefault
	}
	preset.UpdatedAt = time.Now()

	// Update cost-specific fields
	if req.LaborCostPerHour != nil {
		cost.LaborCostPerHour = *req.LaborCostPerHour
	}
	if req.PackagingCostPerItem != nil {
		cost.PackagingCostPerItem = *req.PackagingCostPerItem
	}
	if req.ShippingCostBase != nil {
		cost.ShippingCostBase = *req.ShippingCostBase
	}
	if req.ShippingCostPerGram != nil {
		cost.ShippingCostPerGram = *req.ShippingCostPerGram
	}
	if req.OverheadPercentage != nil {
		cost.OverheadPercentage = *req.OverheadPercentage
	}
	if req.ProfitMarginPercentage != nil {
		cost.ProfitMarginPercentage = *req.ProfitMarginPercentage
	}
	if req.PostProcessingCostPerHour != nil {
		cost.PostProcessingCostPerHour = *req.PostProcessingCostPerHour
	}
	if req.SupportRemovalCostPerHour != nil {
		cost.SupportRemovalCostPerHour = *req.SupportRemovalCostPerHour
	}
	if req.QualityControlCostPerItem != nil {
		cost.QualityControlCostPerItem = *req.QualityControlCostPerItem
	}

	// Validate updated entities
	if err := preset.Validate(); err != nil {
		return nil, err
	}
	if err := cost.Validate(); err != nil {
		return nil, err
	}

	// Save updates
	if err := uc.presetRepo.Update(preset); err != nil {
		return nil, err
	}
	if err := uc.presetRepo.UpdateCost(cost); err != nil {
		return nil, err
	}

	return preset, nil
}
