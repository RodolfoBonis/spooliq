package repositories

import (
	"context"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
)

// PresetRepository defines the contract for preset data persistence
type PresetRepository interface {
	// GetEnergyLocations retrieves all available energy preset locations
	GetEnergyLocations(ctx context.Context) ([]string, error)

	// GetEnergyPresets retrieves energy presets, optionally filtered by location
	GetEnergyPresets(ctx context.Context, location string) ([]*entities.Preset, error)

	// GetMachinePresets retrieves all machine presets
	GetMachinePresets(ctx context.Context) ([]*entities.Preset, error)

	// GetPresetByKey retrieves a preset by its key
	GetPresetByKey(ctx context.Context, key string) (*entities.Preset, error)

	// CreatePreset creates a new preset
	CreatePreset(ctx context.Context, preset *entities.Preset) error

	// UpdatePreset updates an existing preset
	UpdatePreset(ctx context.Context, preset *entities.Preset) error

	// DeletePreset deletes a preset by key
	DeletePreset(ctx context.Context, key string) error

	// GetPresetsByKeyPrefix retrieves presets that start with the given key prefix
	GetPresetsByKeyPrefix(ctx context.Context, keyPrefix string) ([]*entities.Preset, error)
}