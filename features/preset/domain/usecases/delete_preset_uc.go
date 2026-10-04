package usecases

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	"github.com/google/uuid"
)

// DeletePresetUseCase handles deleting presets
type DeletePresetUseCase struct {
	presetRepo repositories.PresetRepository
}

// NewDeletePresetUseCase creates a new instance of DeletePresetUseCase
func NewDeletePresetUseCase(presetRepo repositories.PresetRepository) *DeletePresetUseCase {
	return &DeletePresetUseCase{
		presetRepo: presetRepo,
	}
}

// Execute soft deletes a preset by ID within the organization scope.
// Returns:
//   - gorm.ErrRecordNotFound when the preset does not exist for the organization
//     (cross-organization access is indistinguishable from "not found" by design);
//   - entities.ErrCannotDeleteDefaultPreset when the preset is marked as default.
func (uc *DeletePresetUseCase) Execute(id uuid.UUID, organizationID string) error {
	// Check if preset exists within the organization scope
	preset, err := uc.presetRepo.GetByID(id, organizationID)
	if err != nil {
		return err
	}

	// Default presets cannot be deleted
	if preset.IsDefault {
		return entities.ErrCannotDeleteDefaultPreset
	}

	// Perform soft delete (organization-scoped)
	return uc.presetRepo.Delete(id, organizationID)
}
