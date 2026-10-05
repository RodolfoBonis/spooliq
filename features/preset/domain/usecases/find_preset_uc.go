package usecases

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// FindPresetUseCase handles finding and retrieving presets
type FindPresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewFindPresetUseCase creates a new instance of FindPresetUseCase
func NewFindPresetUseCase(presetRepo repositories.PresetRepository) *FindPresetUseCase {
	return &FindPresetUseCase{
		presetRepo: presetRepo,
	}
}

// MachinePresetResponse is a type alias for the repository response type
type MachinePresetResponse = repositories.MachinePresetResponse

// EnergyPresetResponse is a type alias for the repository response type
type EnergyPresetResponse = repositories.EnergyPresetResponse

// CostPresetResponse is a type alias for the repository response type
type CostPresetResponse = repositories.CostPresetResponse

// FindByID finds a preset by its ID within the organization scope.
func (uc *FindPresetUseCase) FindByID(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	return uc.presetRepo.GetByID(id, organizationID)
}

// FindPresets finds presets for an organization applying the given combinable filters.
func (uc *FindPresetUseCase) FindPresets(organizationID string, filters entities.PresetFilters) ([]*entities.PresetEntity, error) {
	return uc.presetRepo.ListPresets(organizationID, filters)
}

// FindMachinePresetByID finds a machine preset with full details within the organization scope.
func (uc *FindPresetUseCase) FindMachinePresetByID(id uuid.UUID, organizationID string) (*MachinePresetResponse, error) {
	// Get base preset (organization-scoped)
	preset, err := uc.presetRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	// Verify it's a machine preset
	if preset.Type != entities.PresetTypeMachine {
		return nil, entities.ErrInvalidPresetType
	}

	// Get machine-specific data (organization-scoped)
	machine, err := uc.presetRepo.GetMachineByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	return &MachinePresetResponse{
		ID:                     preset.ID.String(),
		Name:                   preset.Name,
		Description:            preset.Description,
		Type:                   string(preset.Type),
		IsActive:               preset.IsActive,
		IsDefault:              preset.IsDefault,
		CreatedAt:              preset.CreatedAt,
		UpdatedAt:              preset.UpdatedAt,
		Brand:                  machine.Brand,
		Model:                  machine.Model,
		BuildVolumeX:           machine.BuildVolumeX,
		BuildVolumeY:           machine.BuildVolumeY,
		BuildVolumeZ:           machine.BuildVolumeZ,
		NozzleDiameter:         machine.NozzleDiameter,
		LayerHeightMin:         machine.LayerHeightMin,
		LayerHeightMax:         machine.LayerHeightMax,
		PrintSpeedMax:          machine.PrintSpeedMax,
		PowerConsumption:       machine.PowerConsumption,
		BedTemperatureMax:      machine.BedTemperatureMax,
		ExtruderTemperatureMax: machine.ExtruderTemperatureMax,
		FilamentDiameter:       machine.FilamentDiameter,
		CostPerHour:            machine.CostPerHour,
	}, nil
}

// FindEnergyPresetByID finds an energy preset with full details within the organization scope.
func (uc *FindPresetUseCase) FindEnergyPresetByID(id uuid.UUID, organizationID string) (*EnergyPresetResponse, error) {
	// Get base preset (organization-scoped)
	preset, err := uc.presetRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	// Verify it's an energy preset
	if preset.Type != entities.PresetTypeEnergy {
		return nil, entities.ErrInvalidPresetType
	}

	// Get energy-specific data (organization-scoped)
	energy, err := uc.presetRepo.GetEnergyByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	return &EnergyPresetResponse{
		ID:                    preset.ID.String(),
		Name:                  preset.Name,
		Description:           preset.Description,
		Type:                  string(preset.Type),
		IsActive:              preset.IsActive,
		IsDefault:             preset.IsDefault,
		CreatedAt:             preset.CreatedAt,
		UpdatedAt:             preset.UpdatedAt,
		Country:               energy.Country,
		State:                 energy.State,
		City:                  energy.City,
		EnergyCostPerKwh:      energy.EnergyCostPerKwh,
		Currency:              energy.Currency,
		Provider:              energy.Provider,
		TariffType:            energy.TariffType,
		PeakHourMultiplier:    energy.PeakHourMultiplier,
		OffPeakHourMultiplier: energy.OffPeakHourMultiplier,
	}, nil
}

// FindCostPresetByID finds a cost preset with full details within the organization scope.
func (uc *FindPresetUseCase) FindCostPresetByID(id uuid.UUID, organizationID string) (*CostPresetResponse, error) {
	// Get base preset (organization-scoped)
	preset, err := uc.presetRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	// Verify it's a cost preset
	if preset.Type != entities.PresetTypeCost {
		return nil, entities.ErrInvalidPresetType
	}

	// Get cost-specific data (organization-scoped)
	cost, err := uc.presetRepo.GetCostByID(id, organizationID)
	if err != nil {
		return nil, err
	}

	return &CostPresetResponse{
		ID:                        preset.ID.String(),
		Name:                      preset.Name,
		Description:               preset.Description,
		Type:                      string(preset.Type),
		IsActive:                  preset.IsActive,
		IsDefault:                 preset.IsDefault,
		CreatedAt:                 preset.CreatedAt,
		UpdatedAt:                 preset.UpdatedAt,
		LaborCostPerHour:          cost.LaborCostPerHour,
		PackagingCostPerItem:      cost.PackagingCostPerItem,
		ShippingCostBase:          cost.ShippingCostBase,
		ShippingCostPerGram:       cost.ShippingCostPerGram,
		OverheadPercentage:        cost.OverheadPercentage,
		ProfitMarginPercentage:    cost.ProfitMarginPercentage,
		PostProcessingCostPerHour: cost.PostProcessingCostPerHour,
		SupportRemovalCostPerHour: cost.SupportRemovalCostPerHour,
		QualityControlCostPerItem: cost.QualityControlCostPerItem,
	}, nil
}

// FindAllMachinePresets finds all machine presets with full details for a specific organization
func (uc *FindPresetUseCase) FindAllMachinePresets(organizationID string) ([]*MachinePresetResponse, error) {
	return uc.presetRepo.GetMachinePresets(organizationID)
}

// FindAllEnergyPresets finds all energy presets with full details for a specific organization
func (uc *FindPresetUseCase) FindAllEnergyPresets(organizationID string) ([]*EnergyPresetResponse, error) {
	return uc.presetRepo.GetEnergyPresets(organizationID)
}

// FindAllCostPresets finds all cost presets with full details for a specific organization
func (uc *FindPresetUseCase) FindAllCostPresets(organizationID string) ([]*CostPresetResponse, error) {
	return uc.presetRepo.GetCostPresets(organizationID)
}

// FindMachinePresetsByBrand finds machine presets by brand for a specific organization
func (uc *FindPresetUseCase) FindMachinePresetsByBrand(brand, organizationID string) ([]*MachinePresetResponse, error) {
	return uc.presetRepo.GetMachinePresetsByBrand(brand, organizationID)
}

// FindEnergyPresetsByLocation finds energy presets by location for a specific organization
func (uc *FindPresetUseCase) FindEnergyPresetsByLocation(country, state, city, organizationID string) ([]*EnergyPresetResponse, error) {
	return uc.presetRepo.GetEnergyPresetsByLocation(country, state, city, organizationID)
}

// FindEnergyPresetsByCurrency finds energy presets by currency for a specific organization
func (uc *FindPresetUseCase) FindEnergyPresetsByCurrency(currency, organizationID string) ([]*EnergyPresetResponse, error) {
	return uc.presetRepo.GetEnergyPresetsByCurrency(currency, organizationID)
}
