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

// MachinePresetResponse represents a machine preset with base info
type MachinePresetResponse struct {
	*entities.PresetEntity
	*entities.MachinePresetEntity
}

// EnergyPresetResponse represents an energy preset with base info
type EnergyPresetResponse struct {
	*entities.PresetEntity
	*entities.EnergyPresetEntity
}

// CostPresetResponse represents a cost preset with base info
type CostPresetResponse struct {
	*entities.PresetEntity
	*entities.CostPresetEntity
}

// FindByID finds a preset by its ID
func (uc *FindPresetUseCase) FindByID(id uuid.UUID) (*entities.PresetEntity, error) {
	return uc.presetRepo.GetByID(id)
}

// FindByType finds presets by type
func (uc *FindPresetUseCase) FindByType(presetType entities.PresetType) ([]*entities.PresetEntity, error) {
	return uc.presetRepo.GetByType(presetType)
}

// FindByUserID finds presets belonging to a specific user
func (uc *FindPresetUseCase) FindByUserID(userID uuid.UUID) ([]*entities.PresetEntity, error) {
	return uc.presetRepo.GetByUserID(userID)
}

// FindGlobalPresets finds all global presets (not user-specific)
func (uc *FindPresetUseCase) FindGlobalPresets() ([]*entities.PresetEntity, error) {
	return uc.presetRepo.GetGlobalPresets()
}

// FindActivePresets finds all active presets
func (uc *FindPresetUseCase) FindActivePresets() ([]*entities.PresetEntity, error) {
	return uc.presetRepo.GetActivePresets()
}

// FindDefaultPresets finds all default presets
func (uc *FindPresetUseCase) FindDefaultPresets() ([]*entities.PresetEntity, error) {
	return uc.presetRepo.GetDefaultPresets()
}

// FindMachinePresetByID finds a machine preset with full details
func (uc *FindPresetUseCase) FindMachinePresetByID(id uuid.UUID) (*MachinePresetResponse, error) {
	// Get base preset
	preset, err := uc.presetRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Verify it's a machine preset
	if preset.Type != entities.PresetTypeMachine {
		return nil, entities.ErrInvalidPresetType
	}

	// Get machine-specific data
	machine, err := uc.presetRepo.GetMachineByID(id)
	if err != nil {
		return nil, err
	}

	return &MachinePresetResponse{
		PresetEntity:        preset,
		MachinePresetEntity: machine,
	}, nil
}

// FindMachinePresetsByBrand finds machine presets by brand
func (uc *FindPresetUseCase) FindMachinePresetsByBrand(brand string) ([]*MachinePresetResponse, error) {
	machines, err := uc.presetRepo.GetMachinesByBrand(brand)
	if err != nil {
		return nil, err
	}

	var responses []*MachinePresetResponse
	for _, machine := range machines {
		// Get base preset for each machine
		preset, err := uc.presetRepo.GetByID(machine.ID)
		if err != nil {
			continue // Skip if base preset not found
		}

		responses = append(responses, &MachinePresetResponse{
			PresetEntity:        preset,
			MachinePresetEntity: machine,
		})
	}

	return responses, nil
}

// FindEnergyPresetByID finds an energy preset with full details
func (uc *FindPresetUseCase) FindEnergyPresetByID(id uuid.UUID) (*EnergyPresetResponse, error) {
	// Get base preset
	preset, err := uc.presetRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Verify it's an energy preset
	if preset.Type != entities.PresetTypeEnergy {
		return nil, entities.ErrInvalidPresetType
	}

	// Get energy-specific data
	energy, err := uc.presetRepo.GetEnergyByID(id)
	if err != nil {
		return nil, err
	}

	return &EnergyPresetResponse{
		PresetEntity:       preset,
		EnergyPresetEntity: energy,
	}, nil
}

// FindEnergyPresetsByLocation finds energy presets by location
func (uc *FindPresetUseCase) FindEnergyPresetsByLocation(country, state, city string) ([]*EnergyPresetResponse, error) {
	energies, err := uc.presetRepo.GetEnergyByLocation(country, state, city)
	if err != nil {
		return nil, err
	}

	var responses []*EnergyPresetResponse
	for _, energy := range energies {
		// Get base preset for each energy
		preset, err := uc.presetRepo.GetByID(energy.ID)
		if err != nil {
			continue // Skip if base preset not found
		}

		responses = append(responses, &EnergyPresetResponse{
			PresetEntity:       preset,
			EnergyPresetEntity: energy,
		})
	}

	return responses, nil
}

// FindEnergyPresetsByCurrency finds energy presets by currency
func (uc *FindPresetUseCase) FindEnergyPresetsByCurrency(currency string) ([]*EnergyPresetResponse, error) {
	energies, err := uc.presetRepo.GetEnergyByCurrency(currency)
	if err != nil {
		return nil, err
	}

	var responses []*EnergyPresetResponse
	for _, energy := range energies {
		// Get base preset for each energy
		preset, err := uc.presetRepo.GetByID(energy.ID)
		if err != nil {
			continue // Skip if base preset not found
		}

		responses = append(responses, &EnergyPresetResponse{
			PresetEntity:       preset,
			EnergyPresetEntity: energy,
		})
	}

	return responses, nil
}

// FindCostPresetByID finds a cost preset with full details
func (uc *FindPresetUseCase) FindCostPresetByID(id uuid.UUID) (*CostPresetResponse, error) {
	// Get base preset
	preset, err := uc.presetRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Verify it's a cost preset
	if preset.Type != entities.PresetTypeCost {
		return nil, entities.ErrInvalidPresetType
	}

	// Get cost-specific data
	cost, err := uc.presetRepo.GetCostByID(id)
	if err != nil {
		return nil, err
	}

	return &CostPresetResponse{
		PresetEntity:     preset,
		CostPresetEntity: cost,
	}, nil
}
