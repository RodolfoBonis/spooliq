// Package entities contains the domain entities and DTOs for the print profile feature.
package entities

import (
	"time"

	"github.com/google/uuid"
)

// ProfileEntity represents a reusable print profile: a named combination of a
// machine preset, an energy preset and an optional cost preset.
type ProfileEntity struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  string     `json:"organization_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description,omitempty"`
	MachinePresetID uuid.UUID  `json:"machine_preset_id"`
	EnergyPresetID  uuid.UUID  `json:"energy_preset_id"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"`
	IsDefault       bool       `json:"is_default"`
	CreatedBy       string     `json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// PresetRef is the embedded {id, name} reference to a preset in a profile response.
type PresetRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ProfileResponse is the API representation of a profile, embedding the {id, name}
// of each referenced preset so the web app does not need extra lookups.
type ProfileResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	IsDefault     bool       `json:"is_default"`
	CreatedBy     string     `json:"created_by,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	MachinePreset PresetRef  `json:"machine_preset"`
	EnergyPreset  PresetRef  `json:"energy_preset"`
	CostPreset    *PresetRef `json:"cost_preset,omitempty"`
}

// CreateProfileRequest is the body for creating a profile. Name is optional and
// auto-generated from the referenced presets when empty.
type CreateProfileRequest struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	MachinePresetID uuid.UUID  `json:"machine_preset_id" binding:"required"`
	EnergyPresetID  uuid.UUID  `json:"energy_preset_id" binding:"required"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id"`
	IsDefault       bool       `json:"is_default"`
}

// UpdateProfileRequest is the body for updating a profile. All fields are
// optional; only provided ones are applied. ID is taken from the URL path.
type UpdateProfileRequest struct {
	ID              uuid.UUID  `json:"-"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	MachinePresetID *uuid.UUID `json:"machine_preset_id"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id"`
	IsDefault       *bool      `json:"is_default"`
}
