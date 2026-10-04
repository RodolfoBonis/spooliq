package mocks

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
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
