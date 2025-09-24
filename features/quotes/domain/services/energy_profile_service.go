package services

import (
	"context"
	"fmt"

	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// EnergyProfileService handles energy profile creation from presets or custom data
type EnergyProfileService interface {
	// CreateEnergyProfileFromRequest creates an EnergyProfile entity from request data
	// Supports both preset reference and custom data
	CreateEnergyProfileFromRequest(ctx context.Context, req *dto.CreateEnergyProfileRequest, ownerUserID string) (*quotesEntities.EnergyProfile, error)
}

// PresetRepository defines the interface for accessing preset data
type PresetRepository interface {
	// GetPresetByKey retrieves a preset by its key
	GetPresetByKey(ctx context.Context, key string) (*presetsEntities.Preset, error)
}

type energyProfileServiceImpl struct {
	presetRepo PresetRepository
}

// NewEnergyProfileService creates a new energy profile service
func NewEnergyProfileService(presetRepo PresetRepository) EnergyProfileService {
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