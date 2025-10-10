package entities

import (
	"errors"

	"github.com/google/uuid"
)

// MachinePresetEntity represents a 3D printer machine preset
type MachinePresetEntity struct {
	ID                     uuid.UUID `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Brand                  string    `json:"brand,omitempty"`
	Model                  string    `json:"model,omitempty"`
	BuildVolumeX           float32   `json:"build_volume_x"`
	BuildVolumeY           float32   `json:"build_volume_y"`
	BuildVolumeZ           float32   `json:"build_volume_z"`
	NozzleDiameter         float32   `json:"nozzle_diameter"`
	LayerHeightMin         float32   `json:"layer_height_min"`
	LayerHeightMax         float32   `json:"layer_height_max"`
	PrintSpeedMax          float32   `json:"print_speed_max"`
	PowerConsumption       float32   `json:"power_consumption"`
	BedTemperatureMax      float32   `json:"bed_temperature_max"`
	ExtruderTemperatureMax float32   `json:"extruder_temperature_max"`
	FilamentDiameter       float32   `json:"filament_diameter"`
	CostPerHour            float32   `json:"cost_per_hour"`
}

// Validate validates the machine preset entity
func (m *MachinePresetEntity) Validate() error {
	if m.BuildVolumeX <= 0 {
		return errors.New("build volume X must be greater than 0")
	}
	if m.BuildVolumeY <= 0 {
		return errors.New("build volume Y must be greater than 0")
	}
	if m.BuildVolumeZ <= 0 {
		return errors.New("build volume Z must be greater than 0")
	}
	if m.NozzleDiameter <= 0 {
		return errors.New("nozzle diameter must be greater than 0")
	}
	if m.LayerHeightMin <= 0 {
		return errors.New("minimum layer height must be greater than 0")
	}
	if m.LayerHeightMax <= 0 {
		return errors.New("maximum layer height must be greater than 0")
	}
	if m.LayerHeightMin >= m.LayerHeightMax {
		return errors.New("minimum layer height must be less than maximum layer height")
	}
	if m.PrintSpeedMax <= 0 {
		return errors.New("maximum print speed must be greater than 0")
	}
	if m.PowerConsumption <= 0 {
		return errors.New("power consumption must be greater than 0")
	}
	if m.FilamentDiameter <= 0 {
		return errors.New("filament diameter must be greater than 0")
	}

	return nil
}

// CalculateVolume calculates the build volume in cubic mm
func (m *MachinePresetEntity) CalculateVolume() float32 {
	return m.BuildVolumeX * m.BuildVolumeY * m.BuildVolumeZ
}

// IsLayerHeightValid checks if a given layer height is valid for this machine
func (m *MachinePresetEntity) IsLayerHeightValid(layerHeight float32) bool {
	return layerHeight >= m.LayerHeightMin && layerHeight <= m.LayerHeightMax
}
