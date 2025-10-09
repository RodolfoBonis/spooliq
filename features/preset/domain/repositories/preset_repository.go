package repositories

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// PresetRepository defines the contract for preset data operations
type PresetRepository interface {
	// Base preset operations
	Create(preset *entities.PresetEntity) error
	GetByID(id uuid.UUID) (*entities.PresetEntity, error)
	GetByType(presetType entities.PresetType) ([]*entities.PresetEntity, error)
	GetByUserID(userID uuid.UUID) ([]*entities.PresetEntity, error)
	GetGlobalPresets() ([]*entities.PresetEntity, error)
	GetActivePresets() ([]*entities.PresetEntity, error)
	GetDefaultPresets() ([]*entities.PresetEntity, error)
	Update(preset *entities.PresetEntity) error
	Delete(id uuid.UUID) error

	// Machine preset operations
	CreateMachine(preset *entities.PresetEntity, machine *entities.MachinePresetEntity) error
	GetMachineByID(id uuid.UUID) (*entities.MachinePresetEntity, error)
	GetMachinesByBrand(brand string) ([]*entities.MachinePresetEntity, error)
	UpdateMachine(machine *entities.MachinePresetEntity) error

	// Energy preset operations
	CreateEnergy(preset *entities.PresetEntity, energy *entities.EnergyPresetEntity) error
	GetEnergyByID(id uuid.UUID) (*entities.EnergyPresetEntity, error)
	GetEnergyByLocation(country, state, city string) ([]*entities.EnergyPresetEntity, error)
	GetEnergyByCurrency(currency string) ([]*entities.EnergyPresetEntity, error)
	UpdateEnergy(energy *entities.EnergyPresetEntity) error

	// Cost preset operations
	CreateCost(preset *entities.PresetEntity, cost *entities.CostPresetEntity) error
	GetCostByID(id uuid.UUID) (*entities.CostPresetEntity, error)
	UpdateCost(cost *entities.CostPresetEntity) error
}
