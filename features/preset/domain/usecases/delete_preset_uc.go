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

// Execute soft deletes a preset by ID
func (uc *DeletePresetUseCase) Execute(id uuid.UUID) error {
	// Check if preset exists
	preset, err := uc.presetRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if it's a default preset - don't allow deletion of defaults
	if preset.IsDefault {
		return entities.ErrInvalidPresetType // Could create a specific error for this
	}

	// Perform soft delete
	return uc.presetRepo.Delete(id)
}
