// Package usecases contains the business logic for the print profile feature.
package usecases

import (
	presetEntities "github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"github.com/google/uuid"
)

// ProfileUseCase implements the print profile business logic. It depends on the
// preset repository to validate that referenced presets belong to the caller's
// organization and have the expected type, and to resolve preset names for
// responses and auto-generated profile names.
type ProfileUseCase struct {
	profileRepo profileRepos.PrintProfileRepository
	presetRepo  presetRepos.PresetRepository
}

// NewProfileUseCase creates a new ProfileUseCase.
func NewProfileUseCase(profileRepo profileRepos.PrintProfileRepository, presetRepo presetRepos.PresetRepository) *ProfileUseCase {
	return &ProfileUseCase{
		profileRepo: profileRepo,
		presetRepo:  presetRepo,
	}
}

// resolvedPresets holds the validated presets referenced by a profile, used both
// for auto-naming and response building.
type resolvedPresets struct {
	machine *presetEntities.PresetEntity
	energy  *presetEntities.PresetEntity
	cost    *presetEntities.PresetEntity
}

// validateRefs verifies the referenced presets exist within the organization and
// have the expected type, returning the loaded presets. A missing or wrong-typed
// reference surfaces a dedicated typed error so the handler can map it to 400.
func (uc *ProfileUseCase) validateRefs(machineID, energyID uuid.UUID, costID *uuid.UUID, organizationID string) (*resolvedPresets, error) {
	machine, err := uc.presetRepo.GetByID(machineID, organizationID)
	if err != nil || machine.Type != presetEntities.PresetTypeMachine {
		return nil, entities.ErrInvalidMachinePreset
	}

	energy, err := uc.presetRepo.GetByID(energyID, organizationID)
	if err != nil || energy.Type != presetEntities.PresetTypeEnergy {
		return nil, entities.ErrInvalidEnergyPreset
	}

	resolved := &resolvedPresets{machine: machine, energy: energy}

	if costID != nil {
		cost, err := uc.presetRepo.GetByID(*costID, organizationID)
		if err != nil || cost.Type != presetEntities.PresetTypeCost {
			return nil, entities.ErrInvalidCostPreset
		}
		resolved.cost = cost
	}

	return resolved, nil
}

// presetRef builds an embedded {id, name} reference, resolving the preset name
// within the organization. When the preset cannot be loaded (e.g. later deleted)
// the name degrades to empty rather than failing the whole response.
func (uc *ProfileUseCase) presetRef(id uuid.UUID, organizationID string) entities.PresetRef {
	ref := entities.PresetRef{ID: id.String()}
	if preset, err := uc.presetRepo.GetByID(id, organizationID); err == nil {
		ref.Name = preset.Name
	}
	return ref
}

// buildResponse assembles the API response for a profile, embedding each
// referenced preset's {id, name}.
func (uc *ProfileUseCase) buildResponse(profile *entities.ProfileEntity) entities.ProfileResponse {
	resp := entities.ProfileResponse{
		ID:            profile.ID.String(),
		Name:          profile.Name,
		Description:   profile.Description,
		IsDefault:     profile.IsDefault,
		CreatedBy:     profile.CreatedBy,
		CreatedAt:     profile.CreatedAt,
		UpdatedAt:     profile.UpdatedAt,
		MachinePreset: uc.presetRef(profile.MachinePresetID, profile.OrganizationID),
		EnergyPreset:  uc.presetRef(profile.EnergyPresetID, profile.OrganizationID),
	}
	if profile.CostPresetID != nil {
		ref := uc.presetRef(*profile.CostPresetID, profile.OrganizationID)
		resp.CostPreset = &ref
	}
	return resp
}
