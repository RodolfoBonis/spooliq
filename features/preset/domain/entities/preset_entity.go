package entities

import (
	"time"

	"github.com/google/uuid"
)

// PresetType defines the type of preset
type PresetType string

const (
	// PresetTypeMachine Preset type machine
	PresetTypeMachine PresetType = "machine"

	// PresetTypeEnergy Preset type energy
	PresetTypeEnergy PresetType = "energy"

	// PresetTypeCost Preset type cost
	PresetTypeCost PresetType = "cost"
)

// PresetEntity represents the base preset entity
type PresetEntity struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	Type           PresetType `json:"type"`
	IsActive       bool       `json:"is_active"`
	IsDefault      bool       `json:"is_default"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	OrganizationID string     `json:"organization_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// IsGlobal returns true if the preset is global (not user-specific)
func (p *PresetEntity) IsGlobal() bool {
	return p.UserID == nil
}

// IsUserSpecific returns true if the preset belongs to a specific user
func (p *PresetEntity) IsUserSpecific() bool {
	return p.UserID != nil
}

// Validate validates the preset entity
func (p *PresetEntity) Validate() error {
	if p.Name == "" {
		return ErrPresetNameRequired
	}

	switch p.Type {
	case PresetTypeMachine, PresetTypeEnergy, PresetTypeCost:
		// Valid types
	default:
		return ErrInvalidPresetType
	}

	return nil
}
