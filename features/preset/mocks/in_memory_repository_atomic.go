package mocks

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The *WithPreset methods mirror the real repository's atomic update contract:
// the base preset and its type-specific child are updated as a single unit. The
// fake validates that BOTH rows exist (in the caller's organization, not
// soft-deleted) before mutating anything, so a cross-org or missing row yields
// gorm.ErrRecordNotFound and leaves all stored data untouched — never a partial
// write.

// UpdateMachineWithPreset atomically updates the base preset and machine child.
func (r *InMemoryPresetRepository) UpdateMachineWithPreset(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error {
	r.record(preset.OrganizationID)
	if r.liveInOrg(preset.ID, preset.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	if _, ok := r.machines[machine.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	presetClone := *preset
	r.presets[preset.ID] = &presetClone
	machineClone := *machine
	r.machines[machine.ID] = &machineClone
	return nil
}

// UpdateEnergyWithPreset atomically updates the base preset and energy child.
func (r *InMemoryPresetRepository) UpdateEnergyWithPreset(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error {
	r.record(preset.OrganizationID)
	if r.liveInOrg(preset.ID, preset.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	if _, ok := r.energies[energy.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	presetClone := *preset
	r.presets[preset.ID] = &presetClone
	energyClone := *energy
	r.energies[energy.ID] = &energyClone
	return nil
}

// UpdateCostWithPreset atomically updates the base preset and cost child.
func (r *InMemoryPresetRepository) UpdateCostWithPreset(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error {
	r.record(preset.OrganizationID)
	if r.liveInOrg(preset.ID, preset.OrganizationID) == nil {
		return gorm.ErrRecordNotFound
	}
	if _, ok := r.costs[cost.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	presetClone := *preset
	r.presets[preset.ID] = &presetClone
	costClone := *cost
	r.costs[cost.ID] = &costClone
	return nil
}

// clearOtherDefaults unsets is_default on every other live preset of the same
// (organization, type), mirroring the real repository's transactional clearing.
func (r *InMemoryPresetRepository) clearOtherDefaults(organizationID string, presetType entities.PresetType, exceptID uuid.UUID) {
	for id, preset := range r.presets {
		if id == exceptID || preset.DeletedAt != nil {
			continue
		}
		if preset.OrganizationID == organizationID && preset.Type == presetType {
			preset.IsDefault = false
		}
	}
}

// SetDefault marks the preset as the single default for its (organization, type)
// pair, clearing any sibling default in the same organization.
func (r *InMemoryPresetRepository) SetDefault(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	r.record(organizationID)
	preset := r.liveInOrg(id, organizationID)
	if preset == nil {
		return nil, gorm.ErrRecordNotFound
	}
	r.clearOtherDefaults(organizationID, preset.Type, id)
	preset.IsDefault = true
	preset.UpdatedAt = time.Now()
	clone := *preset
	return &clone, nil
}

// Duplicate copies a preset (base + child) within the same organization. The
// copy is never a default and takes the given name.
func (r *InMemoryPresetRepository) Duplicate(id uuid.UUID, organizationID string, newName string) (*entities.PresetEntity, error) {
	r.record(organizationID)
	preset := r.liveInOrg(id, organizationID)
	if preset == nil {
		return nil, gorm.ErrRecordNotFound
	}
	now := time.Now()
	newID := uuid.New()
	newPreset := *preset
	newPreset.ID = newID
	newPreset.Name = newName
	newPreset.IsDefault = false
	newPreset.CreatedAt = now
	newPreset.UpdatedAt = now
	newPreset.DeletedAt = nil

	switch preset.Type {
	case entities.PresetTypeMachine:
		child, ok := r.machines[id]
		if !ok {
			return nil, gorm.ErrRecordNotFound
		}
		cc := *child
		cc.ID = newID
		r.Seed(&newPreset, &cc)
	case entities.PresetTypeEnergy:
		child, ok := r.energies[id]
		if !ok {
			return nil, gorm.ErrRecordNotFound
		}
		cc := *child
		cc.ID = newID
		r.Seed(&newPreset, &cc)
	case entities.PresetTypeCost:
		child, ok := r.costs[id]
		if !ok {
			return nil, gorm.ErrRecordNotFound
		}
		cc := *child
		cc.ID = newID
		r.Seed(&newPreset, &cc)
	default:
		return nil, entities.ErrInvalidPresetType
	}

	clone := newPreset
	return &clone, nil
}

// IsReferencedByProfile reports whether the preset id is flagged as referenced by
// a live profile via the ProfileReferenced map (default false).
func (r *InMemoryPresetRepository) IsReferencedByProfile(presetID uuid.UUID, organizationID string) (bool, error) {
	r.record(organizationID)
	return r.ProfileReferenced[presetID], nil
}
