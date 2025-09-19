package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	"github.com/jinzhu/gorm"
)

type presetRepositoryImpl struct {
	db     *gorm.DB
	logger logger.Logger
}

// NewPresetRepository creates a new preset repository implementation
func NewPresetRepository(db *gorm.DB, logger logger.Logger) repositories.PresetRepository {
	return &presetRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// GetEnergyLocations retrieves all available energy preset locations
func (r *presetRepositoryImpl) GetEnergyLocations(ctx context.Context) ([]string, error) {
	var presets []entities.Preset
	err := r.db.Where("key LIKE ?", "energy_%").Find(&presets).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to get energy presets", err)
		return nil, fmt.Errorf("failed to get energy presets: %w", err)
	}

	locationMap := make(map[string]bool)
	var locations []string

	for _, preset := range presets {
		var energyPreset entities.EnergyPreset
		if err := preset.UnmarshalDataTo(&energyPreset); err != nil {
			r.logger.Warning(ctx, "Failed to unmarshal energy preset", map[string]interface{}{
				"preset_key": preset.Key,
				"error":      err.Error(),
			})
			continue
		}

		if energyPreset.Location != "" && !locationMap[energyPreset.Location] {
			locationMap[energyPreset.Location] = true
			locations = append(locations, energyPreset.Location)
		}
	}

	return locations, nil
}

// GetEnergyPresets retrieves energy presets, optionally filtered by location
func (r *presetRepositoryImpl) GetEnergyPresets(ctx context.Context, location string) ([]*entities.Preset, error) {
	query := r.db.Where("key LIKE ?", "energy_%")

	var presets []*entities.Preset
	err := query.Find(&presets).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to get energy presets", err)
		return nil, fmt.Errorf("failed to get energy presets: %w", err)
	}

	// Filter by location if specified
	if location != "" {
		var filteredPresets []*entities.Preset
		for _, preset := range presets {
			var energyPreset entities.EnergyPreset
			if err := preset.UnmarshalDataTo(&energyPreset); err != nil {
				continue
			}
			if strings.EqualFold(energyPreset.Location, location) {
				filteredPresets = append(filteredPresets, preset)
			}
		}
		return filteredPresets, nil
	}

	return presets, nil
}

// GetMachinePresets retrieves all machine presets
func (r *presetRepositoryImpl) GetMachinePresets(ctx context.Context) ([]*entities.Preset, error) {
	var presets []*entities.Preset
	err := r.db.Where("key LIKE ?", "machine_%").Find(&presets).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to get machine presets", err)
		return nil, fmt.Errorf("failed to get machine presets: %w", err)
	}

	return presets, nil
}

// GetPresetByKey retrieves a preset by its key
func (r *presetRepositoryImpl) GetPresetByKey(ctx context.Context, key string) (*entities.Preset, error) {
	var preset entities.Preset
	err := r.db.Where("key = ?", key).First(&preset).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, fmt.Errorf("preset not found: %s", key)
		}
		r.logger.LogError(ctx, "Failed to get preset by key", err)
		return nil, fmt.Errorf("failed to get preset: %w", err)
	}

	return &preset, nil
}

// CreatePreset creates a new preset
func (r *presetRepositoryImpl) CreatePreset(ctx context.Context, preset *entities.Preset) error {
	// Check if preset with same key already exists
	var existingPreset entities.Preset
	err := r.db.Where("key = ?", preset.Key).First(&existingPreset).Error
	if err == nil {
		return fmt.Errorf("preset with key '%s' already exists", preset.Key)
	}
	if !gorm.IsRecordNotFoundError(err) {
		r.logger.LogError(ctx, "Failed to check existing preset", err)
		return fmt.Errorf("failed to check existing preset: %w", err)
	}

	err = r.db.Create(preset).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to create preset", err)
		return fmt.Errorf("failed to create preset: %w", err)
	}

	r.logger.Info(ctx, "Preset created successfully", map[string]interface{}{
		"preset_key": preset.Key,
	})

	return nil
}

// UpdatePreset updates an existing preset
func (r *presetRepositoryImpl) UpdatePreset(ctx context.Context, preset *entities.Preset) error {
	err := r.db.Save(preset).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to update preset", err)
		return fmt.Errorf("failed to update preset: %w", err)
	}

	r.logger.Info(ctx, "Preset updated successfully", map[string]interface{}{
		"preset_key": preset.Key,
	})

	return nil
}

// DeletePreset deletes a preset by key
func (r *presetRepositoryImpl) DeletePreset(ctx context.Context, key string) error {
	result := r.db.Where("key = ?", key).Delete(&entities.Preset{})
	if result.Error != nil {
		r.logger.LogError(ctx, "Failed to delete preset", result.Error)
		return fmt.Errorf("failed to delete preset: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("preset not found: %s", key)
	}

	r.logger.Info(ctx, "Preset deleted successfully", map[string]interface{}{
		"preset_key": key,
	})

	return nil
}

// GetPresetsByKeyPrefix retrieves presets that start with the given key prefix
func (r *presetRepositoryImpl) GetPresetsByKeyPrefix(ctx context.Context, keyPrefix string) ([]*entities.Preset, error) {
	var presets []*entities.Preset
	err := r.db.Where("key LIKE ?", keyPrefix+"%").Find(&presets).Error
	if err != nil {
		r.logger.LogError(ctx, "Failed to get presets by key prefix", err)
		return nil, fmt.Errorf("failed to get presets: %w", err)
	}

	return presets, nil
}
