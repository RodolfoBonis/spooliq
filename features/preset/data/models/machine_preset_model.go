package models

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// MachinePresetModel represents a 3D printer machine preset in the database
type MachinePresetModel struct {
	ID                     uuid.UUID `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	Brand                  string    `gorm:"type:varchar(255)" json:"brand,omitempty"`
	Model                  string    `gorm:"type:varchar(255)" json:"model,omitempty"`
	BuildVolumeX           float32   `gorm:"type:float" json:"build_volume_x"`
	BuildVolumeY           float32   `gorm:"type:float" json:"build_volume_y"`
	BuildVolumeZ           float32   `gorm:"type:float" json:"build_volume_z"`
	NozzleDiameter         float32   `gorm:"type:float" json:"nozzle_diameter"`
	LayerHeightMin         float32   `gorm:"type:float" json:"layer_height_min"`
	LayerHeightMax         float32   `gorm:"type:float" json:"layer_height_max"`
	PrintSpeedMax          float32   `gorm:"type:float" json:"print_speed_max"`
	PowerConsumption       float32   `gorm:"type:float" json:"power_consumption"`
	BedTemperatureMax      float32   `gorm:"type:float" json:"bed_temperature_max"`
	ExtruderTemperatureMax float32   `gorm:"type:float" json:"extruder_temperature_max"`
	FilamentDiameter       float32   `gorm:"type:float" json:"filament_diameter"`
	CostPerHour            float32   `gorm:"type:float" json:"cost_per_hour"`
}

// TableName returns the table name for the machine preset model
func (m *MachinePresetModel) TableName() string { return "machine_presets" }

// FromEntity populates the MachinePresetModel from a MachinePresetEntity
func (m *MachinePresetModel) FromEntity(entity *entities.MachinePresetEntity) {
	m.ID = entity.ID
	m.Brand = entity.Brand
	m.Model = entity.Model
	m.BuildVolumeX = entity.BuildVolumeX
	m.BuildVolumeY = entity.BuildVolumeY
	m.BuildVolumeZ = entity.BuildVolumeZ
	m.NozzleDiameter = entity.NozzleDiameter
	m.LayerHeightMin = entity.LayerHeightMin
	m.LayerHeightMax = entity.LayerHeightMax
	m.PrintSpeedMax = entity.PrintSpeedMax
	m.PowerConsumption = entity.PowerConsumption
	m.BedTemperatureMax = entity.BedTemperatureMax
	m.ExtruderTemperatureMax = entity.ExtruderTemperatureMax
	m.FilamentDiameter = entity.FilamentDiameter
	m.CostPerHour = entity.CostPerHour
}

// ToEntity converts the MachinePresetModel to a MachinePresetEntity
func (m *MachinePresetModel) ToEntity() entities.MachinePresetEntity {
	return entities.MachinePresetEntity{
		ID:                     m.ID,
		Brand:                  m.Brand,
		Model:                  m.Model,
		BuildVolumeX:           m.BuildVolumeX,
		BuildVolumeY:           m.BuildVolumeY,
		BuildVolumeZ:           m.BuildVolumeZ,
		NozzleDiameter:         m.NozzleDiameter,
		LayerHeightMin:         m.LayerHeightMin,
		LayerHeightMax:         m.LayerHeightMax,
		PrintSpeedMax:          m.PrintSpeedMax,
		PowerConsumption:       m.PowerConsumption,
		BedTemperatureMax:      m.BedTemperatureMax,
		ExtruderTemperatureMax: m.ExtruderTemperatureMax,
		FilamentDiameter:       m.FilamentDiameter,
		CostPerHour:            m.CostPerHour,
	}
}
