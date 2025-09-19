package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/services"
	"github.com/go-playground/validator/v10"
)

type presetServiceImpl struct {
	presetRepo repositories.PresetRepository
	logger     logger.Logger
	validator  *validator.Validate
}

// NewPresetService creates a new preset service implementation
func NewPresetService(
	presetRepo repositories.PresetRepository,
	logger logger.Logger,
) services.PresetService {
	return &presetServiceImpl{
		presetRepo: presetRepo,
		logger:     logger,
		validator:  validator.New(),
	}
}

// GetEnergyLocations retrieves all available energy locations
func (s *presetServiceImpl) GetEnergyLocations(ctx context.Context) ([]string, error) {
	locations, err := s.presetRepo.GetEnergyLocations(ctx)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get energy locations", err)
		return nil, fmt.Errorf("failed to get energy locations: %w", err)
	}

	return locations, nil
}

// GetMachinePresets retrieves all machine presets with parsed data
func (s *presetServiceImpl) GetMachinePresets(ctx context.Context) ([]*entities.MachinePreset, error) {
	presets, err := s.presetRepo.GetMachinePresets(ctx)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get machine presets", err)
		return nil, fmt.Errorf("failed to get machine presets: %w", err)
	}

	var machinePresets []*entities.MachinePreset
	for _, preset := range presets {
		var machinePreset entities.MachinePreset
		if err := preset.UnmarshalDataTo(&machinePreset); err != nil {
			s.logger.Warning(ctx, "Failed to unmarshal machine preset", map[string]interface{}{
				"preset_key": preset.Key,
				"error":      err.Error(),
			})
			continue
		}

		machinePresets = append(machinePresets, &machinePreset)
	}

	return machinePresets, nil
}

// GetEnergyPresets retrieves energy presets, optionally filtered by location
func (s *presetServiceImpl) GetEnergyPresets(ctx context.Context, location string) ([]*entities.EnergyPreset, error) {
	presets, err := s.presetRepo.GetEnergyPresets(ctx, location)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get energy presets", err)
		return nil, fmt.Errorf("failed to get energy presets: %w", err)
	}

	var energyPresets []*entities.EnergyPreset
	for _, preset := range presets {
		var energyPreset entities.EnergyPreset
		if err := preset.UnmarshalDataTo(&energyPreset); err != nil {
			s.logger.Warning(ctx, "Failed to unmarshal energy preset", map[string]interface{}{
				"preset_key": preset.Key,
				"error":      err.Error(),
			})
			continue
		}

		energyPresets = append(energyPresets, &energyPreset)
	}

	return energyPresets, nil
}

// CreateEnergyPreset creates a new energy preset (admin only)
func (s *presetServiceImpl) CreateEnergyPreset(ctx context.Context, preset *entities.EnergyPreset, requesterID string) error {
	// Validate admin permissions
	if err := s.ValidateAdminPermissions(ctx, requesterID); err != nil {
		return err
	}

	// Validate preset data
	if err := s.validator.Struct(preset); err != nil {
		s.logger.LogError(ctx, "Energy preset validation failed", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate unique key
	key := s.generateEnergyPresetKey(preset)

	// Create preset entity
	presetEntity := &entities.Preset{
		Key: key,
	}

	if err := presetEntity.MarshalDataFrom(preset); err != nil {
		s.logger.LogError(ctx, "Failed to marshal energy preset data", err)
		return fmt.Errorf("failed to marshal preset data: %w", err)
	}

	// Create in repository
	if err := s.presetRepo.CreatePreset(ctx, presetEntity); err != nil {
		s.logger.LogError(ctx, "Failed to create energy preset", err)
		return fmt.Errorf("failed to create energy preset: %w", err)
	}

	s.logger.Info(ctx, "Energy preset created successfully", map[string]interface{}{
		"preset_key":   key,
		"location":     preset.Location,
		"year":         preset.Year,
		"requester_id": requesterID,
	})

	return nil
}

// CreateMachinePreset creates a new machine preset (admin only)
func (s *presetServiceImpl) CreateMachinePreset(ctx context.Context, preset *entities.MachinePreset, requesterID string) error {
	// Validate admin permissions
	if err := s.ValidateAdminPermissions(ctx, requesterID); err != nil {
		return err
	}

	// Validate preset data
	if err := s.validator.Struct(preset); err != nil {
		s.logger.LogError(ctx, "Machine preset validation failed", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Generate unique key
	key := s.generateMachinePresetKey(preset)

	// Create preset entity
	presetEntity := &entities.Preset{
		Key: key,
	}

	if err := presetEntity.MarshalDataFrom(preset); err != nil {
		s.logger.LogError(ctx, "Failed to marshal machine preset data", err)
		return fmt.Errorf("failed to marshal preset data: %w", err)
	}

	// Create in repository
	if err := s.presetRepo.CreatePreset(ctx, presetEntity); err != nil {
		s.logger.LogError(ctx, "Failed to create machine preset", err)
		return fmt.Errorf("failed to create machine preset: %w", err)
	}

	s.logger.Info(ctx, "Machine preset created successfully", map[string]interface{}{
		"preset_key":   key,
		"name":         preset.Name,
		"brand":        preset.Brand,
		"model":        preset.Model,
		"requester_id": requesterID,
	})

	return nil
}

// UpdatePreset updates an existing preset (admin only)
func (s *presetServiceImpl) UpdatePreset(ctx context.Context, key string, data interface{}, requesterID string) error {
	// Validate admin permissions
	if err := s.ValidateAdminPermissions(ctx, requesterID); err != nil {
		return err
	}

	// Get existing preset
	preset, err := s.presetRepo.GetPresetByKey(ctx, key)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get preset for update", err)
		return fmt.Errorf("failed to get preset: %w", err)
	}

	// Validate new data
	if err := s.validator.Struct(data); err != nil {
		s.logger.LogError(ctx, "Preset update data validation failed", err)
		return fmt.Errorf("validation failed: %w", err)
	}

	// Marshal new data
	if err := preset.MarshalDataFrom(data); err != nil {
		s.logger.LogError(ctx, "Failed to marshal updated preset data", err)
		return fmt.Errorf("failed to marshal preset data: %w", err)
	}

	// Update in repository
	if err := s.presetRepo.UpdatePreset(ctx, preset); err != nil {
		s.logger.LogError(ctx, "Failed to update preset", err)
		return fmt.Errorf("failed to update preset: %w", err)
	}

	s.logger.Info(ctx, "Preset updated successfully", map[string]interface{}{
		"preset_key":   key,
		"requester_id": requesterID,
	})

	return nil
}

// DeletePreset deletes a preset (admin only)
func (s *presetServiceImpl) DeletePreset(ctx context.Context, key string, requesterID string) error {
	// Validate admin permissions
	if err := s.ValidateAdminPermissions(ctx, requesterID); err != nil {
		return err
	}

	// Delete from repository
	if err := s.presetRepo.DeletePreset(ctx, key); err != nil {
		s.logger.LogError(ctx, "Failed to delete preset", err)
		return fmt.Errorf("failed to delete preset: %w", err)
	}

	s.logger.Info(ctx, "Preset deleted successfully", map[string]interface{}{
		"preset_key":   key,
		"requester_id": requesterID,
	})

	return nil
}

// ValidateAdminPermissions validates if user has admin permissions
// TODO: Implement proper admin validation using middleware or user service
func (s *presetServiceImpl) ValidateAdminPermissions(ctx context.Context, userID string) error {
	// For now, we'll implement this validation at the handler level via middleware
	// This is a simplified placeholder to avoid circular dependencies
	if userID == "" {
		return fmt.Errorf("admin permissions required")
	}
	return nil
}

// Helper methods

func (s *presetServiceImpl) generateEnergyPresetKey(preset *entities.EnergyPreset) string {
	// Generate key like: energy_maceio_al_2025
	location := strings.ToLower(strings.ReplaceAll(preset.Location, " ", "_"))
	location = strings.ReplaceAll(location, "-", "_")
	return fmt.Sprintf("energy_%s_%d", location, preset.Year)
}

func (s *presetServiceImpl) generateMachinePresetKey(preset *entities.MachinePreset) string {
	// Generate key like: machine_ender3_v2_creality
	name := strings.ToLower(strings.ReplaceAll(preset.Name, " ", "_"))
	brand := strings.ToLower(strings.ReplaceAll(preset.Brand, " ", "_"))
	model := strings.ToLower(strings.ReplaceAll(preset.Model, " ", "_"))

	timestamp := time.Now().Unix()
	return fmt.Sprintf("machine_%s_%s_%s_%d", name, brand, model, timestamp)
}
