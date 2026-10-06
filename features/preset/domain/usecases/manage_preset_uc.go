package usecases

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// duplicateNameSuffix is appended to a preset's name when it is duplicated.
const duplicateNameSuffix = " (cópia)"

// ManagePresetUseCase handles preset lifecycle actions that are neither plain
// create/update nor delete: setting the default and duplicating.
type ManagePresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewManagePresetUseCase creates a new instance of ManagePresetUseCase.
func NewManagePresetUseCase(presetRepo repositories.PresetRepository) *ManagePresetUseCase {
	return &ManagePresetUseCase{presetRepo: presetRepo}
}

// SetDefault marks the preset as the single default for its (organization, type)
// pair, clearing any sibling default in the same transaction.
func (uc *ManagePresetUseCase) SetDefault(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	return uc.presetRepo.SetDefault(id, organizationID)
}

// Duplicate copies a preset (base + typed child) within the same organization.
// The copy's name is "<name> (cópia)" and it is never a default.
func (uc *ManagePresetUseCase) Duplicate(id uuid.UUID, organizationID string) (*entities.PresetEntity, error) {
	preset, err := uc.presetRepo.GetByID(id, organizationID)
	if err != nil {
		return nil, err
	}
	return uc.presetRepo.Duplicate(id, organizationID, preset.Name+duplicateNameSuffix)
}

// SuggestNameRequest is the input for name suggestion: a preset type plus the
// subset of fields relevant to that type. Unused fields are ignored.
type SuggestNameRequest struct {
	Type entities.PresetType `json:"type" binding:"required"`

	// Machine
	Brand          string  `json:"brand"`
	Model          string  `json:"model"`
	NozzleDiameter float32 `json:"nozzle_diameter"`

	// Energy
	Provider         string  `json:"provider"`
	City             string  `json:"city"`
	State            string  `json:"state"`
	EnergyCostPerKwh float32 `json:"energy_cost_per_kwh"`

	// Cost
	LaborCostPerHour       float32 `json:"labor_cost_per_hour"`
	ProfitMarginPercentage float32 `json:"profit_margin_percentage"`
}

// SuggestName returns an auto-generated name for the given type and fields using
// the same pure functions applied at create time. Returns ErrInvalidPresetType
// for an unknown type.
func (uc *ManagePresetUseCase) SuggestName(req *SuggestNameRequest) (string, error) {
	switch req.Type {
	case entities.PresetTypeMachine:
		return entities.GenerateMachineName(req.Brand, req.Model, req.NozzleDiameter), nil
	case entities.PresetTypeEnergy:
		return entities.GenerateEnergyName(req.Provider, req.City, req.State, req.EnergyCostPerKwh), nil
	case entities.PresetTypeCost:
		return entities.GenerateCostName(req.LaborCostPerHour, req.ProfitMarginPercentage), nil
	default:
		return "", entities.ErrInvalidPresetType
	}
}
