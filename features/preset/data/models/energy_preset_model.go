package models

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// EnergyPresetModel represents an energy cost preset by location in the database
type EnergyPresetModel struct {
	ID                    uuid.UUID `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	OrganizationID        string    `gorm:"type:varchar(255);not null;index:idx_preset_org" json:"organization_id"`
	Country               string    `gorm:"type:varchar(100)" json:"country,omitempty"`
	State                 string    `gorm:"type:varchar(100)" json:"state,omitempty"`
	City                  string    `gorm:"type:varchar(100)" json:"city,omitempty"`
	EnergyCostPerKwh      float32   `gorm:"type:float;not null" json:"energy_cost_per_kwh"`
	Currency              string    `gorm:"type:varchar(3);not null" json:"currency"`
	Provider              string    `gorm:"type:varchar(255)" json:"provider,omitempty"`
	TariffType            string    `gorm:"type:varchar(50)" json:"tariff_type,omitempty"`
	PeakHourMultiplier    float32   `gorm:"type:float;default:1.0" json:"peak_hour_multiplier"`
	OffPeakHourMultiplier float32   `gorm:"type:float;default:1.0" json:"off_peak_hour_multiplier"`
}

// TableName returns the table name for the energy preset model
func (e *EnergyPresetModel) TableName() string { return "energy_presets" }

// FromEntity populates the EnergyPresetModel from an EnergyPresetEntity
func (e *EnergyPresetModel) FromEntity(entity *entities.EnergyPresetEntity) {
	e.ID = entity.ID
	e.OrganizationID = entity.OrganizationID
	e.Country = entity.Country
	e.State = entity.State
	e.City = entity.City
	e.EnergyCostPerKwh = entity.EnergyCostPerKwh
	e.Currency = entity.Currency
	e.Provider = entity.Provider
	e.TariffType = entity.TariffType
	e.PeakHourMultiplier = entity.PeakHourMultiplier
	e.OffPeakHourMultiplier = entity.OffPeakHourMultiplier
}

// ToEntity converts the EnergyPresetModel to an EnergyPresetEntity
func (e *EnergyPresetModel) ToEntity() entities.EnergyPresetEntity {
	return entities.EnergyPresetEntity{
		ID:                    e.ID,
		Country:               e.Country,
		State:                 e.State,
		City:                  e.City,
		EnergyCostPerKwh:      e.EnergyCostPerKwh,
		Currency:              e.Currency,
		Provider:              e.Provider,
		TariffType:            e.TariffType,
		PeakHourMultiplier:    e.PeakHourMultiplier,
		OffPeakHourMultiplier: e.OffPeakHourMultiplier,
	}
}
