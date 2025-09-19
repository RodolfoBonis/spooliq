package services

import (
	"context"
	"github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
)

// PresetService defines the business logic interface for preset operations
type PresetService interface {
	// GetEnergyLocations retrieves all available energy locations
	GetEnergyLocations(ctx context.Context) ([]string, error)

	// GetMachinePresets retrieves all machine presets with parsed data
	GetMachinePresets(ctx context.Context) ([]*entities.MachinePreset, error)

	// GetEnergyPresets retrieves energy presets, optionally filtered by location
	GetEnergyPresets(ctx context.Context, location string) ([]*entities.EnergyPreset, error)

	// CreateEnergyPreset creates a new energy preset (admin only)
	CreateEnergyPreset(ctx context.Context, preset *entities.EnergyPreset, requesterID string) error

	// CreateMachinePreset creates a new machine preset (admin only)
	CreateMachinePreset(ctx context.Context, preset *entities.MachinePreset, requesterID string) error

	// UpdatePreset updates an existing preset (admin only)
	UpdatePreset(ctx context.Context, key string, data interface{}, requesterID string) error

	// DeletePreset deletes a preset (admin only)
	DeletePreset(ctx context.Context, key string, requesterID string) error

	// ValidateAdminPermissions validates if user has admin permissions
	ValidateAdminPermissions(ctx context.Context, userID string) error
}