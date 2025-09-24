package services

import (
	"context"
	"fmt"

	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	presetsRepos "github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// EnergyProfileService handles energy profile creation from presets or custom data
type EnergyProfileService interface {
	// CreateEnergyProfileFromRequest creates an EnergyProfile entity from request data
	// Supports both preset reference and custom data
	CreateEnergyProfileFromRequest(ctx context.Context, req *dto.CreateEnergyProfileRequest, ownerUserID string) (*quotesEntities.EnergyProfile, error)
}

// MachineProfileService handles machine profile creation from presets or custom data
type MachineProfileService interface {
	// CreateMachineProfileFromRequest creates a MachineProfile entity from request data
	// Supports both preset reference and custom data
	CreateMachineProfileFromRequest(ctx context.Context, req *dto.CreateMachineProfileRequest) (*quotesEntities.MachineProfile, error)
}

// CostProfileService handles cost profile creation from presets or custom data
type CostProfileService interface {
	// CreateCostProfileFromRequest creates a CostProfile entity from request data
	// Supports both preset reference and custom data
	CreateCostProfileFromRequest(ctx context.Context, req *dto.CreateCostProfileRequest) (*quotesEntities.CostProfile, error)
}

// MarginProfileService handles margin profile creation from presets or custom data
type MarginProfileService interface {
	// CreateMarginProfileFromRequest creates a MarginProfile entity from request data
	// Supports both preset reference and custom data
	CreateMarginProfileFromRequest(ctx context.Context, req *dto.CreateMarginProfileRequest) (*quotesEntities.MarginProfile, error)
}

type energyProfileServiceImpl struct {
	presetRepo presetsRepos.PresetRepository
}

// NewEnergyProfileService creates a new energy profile service
func NewEnergyProfileService(presetRepo presetsRepos.PresetRepository) EnergyProfileService {
	return &energyProfileServiceImpl{
		presetRepo: presetRepo,
	}
}

// CreateEnergyProfileFromRequest creates an EnergyProfile entity from request data
func (s *energyProfileServiceImpl) CreateEnergyProfileFromRequest(ctx context.Context, req *dto.CreateEnergyProfileRequest, ownerUserID string) (*quotesEntities.EnergyProfile, error) {
	// First validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid energy profile request: %w", err)
	}

	var profile quotesEntities.EnergyProfile

	// Option 1: Load from preset
	if req.PresetKey != "" {
		preset, err := s.presetRepo.GetPresetByKey(ctx, req.PresetKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get energy preset '%s': %w", req.PresetKey, err)
		}

		// Parse preset data as EnergyPreset
		var energyPreset presetsEntities.EnergyPreset
		if err := preset.UnmarshalDataTo(&energyPreset); err != nil {
			return nil, fmt.Errorf("preset '%s' is not a valid energy preset: %w", req.PresetKey, err)
		}

		// Map preset data to profile
		profile.Name = fmt.Sprintf("%s %d", energyPreset.Location, energyPreset.Year)
		profile.BaseTariff = energyPreset.BaseTariff
		profile.FlagSurcharge = energyPreset.FlagSurcharge
		profile.Location = energyPreset.Location
		profile.Year = energyPreset.Year
		profile.Description = energyPreset.Description
	} else {
		// Option 2: Use custom data
		profile.BaseTariff = req.BaseTariff
		profile.FlagSurcharge = req.FlagSurcharge
		profile.Location = req.Location
		profile.Year = req.Year
		profile.Description = req.Description

		// Auto-generate name if not provided
		if req.Name != "" {
			profile.Name = req.Name
		} else {
			profile.Name = fmt.Sprintf("%s %d", req.Location, req.Year)
		}
	}

	// Set owner
	profile.OwnerUserID = &ownerUserID

	return &profile, nil
}

// Machine Profile Service Implementation

type machineProfileServiceImpl struct {
	presetRepo presetsRepos.PresetRepository
}

// NewMachineProfileService creates a new machine profile service
func NewMachineProfileService(presetRepo presetsRepos.PresetRepository) MachineProfileService {
	return &machineProfileServiceImpl{
		presetRepo: presetRepo,
	}
}

// CreateMachineProfileFromRequest creates a MachineProfile entity from request data
func (s *machineProfileServiceImpl) CreateMachineProfileFromRequest(ctx context.Context, req *dto.CreateMachineProfileRequest) (*quotesEntities.MachineProfile, error) {
	// First validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid machine profile request: %w", err)
	}

	var profile quotesEntities.MachineProfile

	// Option 1: Load from preset
	if req.PresetKey != "" {
		preset, err := s.presetRepo.GetPresetByKey(ctx, req.PresetKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine preset '%s': %w", req.PresetKey, err)
		}

		// Parse preset data as MachinePreset
		var machinePreset presetsEntities.MachinePreset
		if err := preset.UnmarshalDataTo(&machinePreset); err != nil {
			return nil, fmt.Errorf("preset '%s' is not a valid machine preset: %w", req.PresetKey, err)
		}

		// Map preset data to profile
		profile.Name = machinePreset.Name
		profile.Brand = machinePreset.Brand
		profile.Model = machinePreset.Model
		profile.Watt = machinePreset.Watt
		profile.IdleFactor = machinePreset.IdleFactor
		profile.Description = machinePreset.Description
		profile.URL = machinePreset.URL
	} else {
		// Option 2: Use custom data
		profile.Name = req.Name
		profile.Brand = req.Brand
		profile.Model = req.Model
		profile.Watt = req.Watt
		profile.IdleFactor = req.IdleFactor
		profile.Description = req.Description
		profile.URL = req.URL
	}

	return &profile, nil
}

// Cost Profile Service Implementation

type costProfileServiceImpl struct {
	presetRepo presetsRepos.PresetRepository
}

// NewCostProfileService creates a new cost profile service
func NewCostProfileService(presetRepo presetsRepos.PresetRepository) CostProfileService {
	return &costProfileServiceImpl{
		presetRepo: presetRepo,
	}
}

// CreateCostProfileFromRequest creates a CostProfile entity from request data
func (s *costProfileServiceImpl) CreateCostProfileFromRequest(ctx context.Context, req *dto.CreateCostProfileRequest) (*quotesEntities.CostProfile, error) {
	// First validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cost profile request: %w", err)
	}

	var profile quotesEntities.CostProfile

	// Option 1: Load from preset
	if req.PresetKey != "" {
		preset, err := s.presetRepo.GetPresetByKey(ctx, req.PresetKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get cost preset '%s': %w", req.PresetKey, err)
		}

		// Parse preset data as CostPreset
		var costPreset presetsEntities.CostPreset
		if err := preset.UnmarshalDataTo(&costPreset); err != nil {
			return nil, fmt.Errorf("preset '%s' is not a valid cost preset: %w", req.PresetKey, err)
		}

		// Map preset data to profile
		profile.Name = costPreset.Name
		profile.WearPercentage = costPreset.WearPercentage
		profile.OverheadAmount = costPreset.OverheadAmount
		profile.Description = costPreset.Description
	} else {
		// Option 2: Use custom data
		profile.WearPercentage = req.WearPercentage
		profile.OverheadAmount = req.OverheadAmount
		profile.Description = req.Description

		// Auto-generate name if not provided
		if req.Name != "" {
			profile.Name = req.Name
		} else {
			profile.Name = fmt.Sprintf("Custom Cost Profile - Wear %.1f%% Overhead %.2f", req.WearPercentage, req.OverheadAmount)
		}
	}

	return &profile, nil
}

// Margin Profile Service Implementation

type marginProfileServiceImpl struct {
	presetRepo presetsRepos.PresetRepository
}

// NewMarginProfileService creates a new margin profile service
func NewMarginProfileService(presetRepo presetsRepos.PresetRepository) MarginProfileService {
	return &marginProfileServiceImpl{
		presetRepo: presetRepo,
	}
}

// CreateMarginProfileFromRequest creates a MarginProfile entity from request data
func (s *marginProfileServiceImpl) CreateMarginProfileFromRequest(ctx context.Context, req *dto.CreateMarginProfileRequest) (*quotesEntities.MarginProfile, error) {
	// First validate the request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid margin profile request: %w", err)
	}

	var profile quotesEntities.MarginProfile

	// Option 1: Load from preset
	if req.PresetKey != "" {
		preset, err := s.presetRepo.GetPresetByKey(ctx, req.PresetKey)
		if err != nil {
			return nil, fmt.Errorf("failed to get margin preset '%s': %w", req.PresetKey, err)
		}

		// Parse preset data as MarginPreset
		var marginPreset presetsEntities.MarginPreset
		if err := preset.UnmarshalDataTo(&marginPreset); err != nil {
			return nil, fmt.Errorf("preset '%s' is not a valid margin preset: %w", req.PresetKey, err)
		}

		// Map preset data to profile
		profile.Name = marginPreset.Name
		profile.PrintingOnlyMargin = marginPreset.PrintingOnlyMargin
		profile.PrintingPlusMargin = marginPreset.PrintingPlusMargin
		profile.FullServiceMargin = marginPreset.FullServiceMargin
		profile.OperatorRatePerHour = marginPreset.OperatorRatePerHour
		profile.ModelerRatePerHour = marginPreset.ModelerRatePerHour
		profile.Description = marginPreset.Description
	} else {
		// Option 2: Use custom data
		profile.PrintingOnlyMargin = req.PrintingOnlyMargin
		profile.PrintingPlusMargin = req.PrintingPlusMargin
		profile.FullServiceMargin = req.FullServiceMargin
		profile.OperatorRatePerHour = req.OperatorRatePerHour
		profile.ModelerRatePerHour = req.ModelerRatePerHour
		profile.Description = req.Description

		// Auto-generate name if not provided
		if req.Name != "" {
			profile.Name = req.Name
		} else {
			profile.Name = fmt.Sprintf("Custom Margin Profile - Print %.1f%% Service %.1f%%", req.PrintingOnlyMargin, req.FullServiceMargin)
		}
	}

	return &profile, nil
}
