package entities

import (
	"errors"

	"github.com/google/uuid"
)

// EnergyPresetEntity represents energy cost configuration by location
type EnergyPresetEntity struct {
	ID                    uuid.UUID `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Country               string    `json:"country,omitempty"`
	State                 string    `json:"state,omitempty"`
	City                  string    `json:"city,omitempty"`
	EnergyCostPerKwh      float32   `json:"energy_cost_per_kwh"`
	Currency              string    `json:"currency"`
	Provider              string    `json:"provider,omitempty"`
	TariffType            string    `json:"tariff_type,omitempty"`
	PeakHourMultiplier    float32   `json:"peak_hour_multiplier"`
	OffPeakHourMultiplier float32   `json:"off_peak_hour_multiplier"`
}

// Validate validates the energy preset entity
func (e *EnergyPresetEntity) Validate() error {
	if e.EnergyCostPerKwh <= 0 {
		return errors.New("energy cost per kWh must be greater than 0")
	}
	if e.Currency == "" {
		return errors.New("currency is required")
	}
	if len(e.Currency) != 3 {
		return errors.New("currency must be a 3-letter ISO code")
	}
	if e.PeakHourMultiplier <= 0 {
		return errors.New("peak hour multiplier must be greater than 0")
	}
	if e.OffPeakHourMultiplier <= 0 {
		return errors.New("off-peak hour multiplier must be greater than 0")
	}

	return nil
}

// CalculatePeakCost calculates the cost per kWh during peak hours
func (e *EnergyPresetEntity) CalculatePeakCost() float32 {
	return e.EnergyCostPerKwh * e.PeakHourMultiplier
}

// CalculateOffPeakCost calculates the cost per kWh during off-peak hours
func (e *EnergyPresetEntity) CalculateOffPeakCost() float32 {
	return e.EnergyCostPerKwh * e.OffPeakHourMultiplier
}

// GetLocationString returns a formatted location string
func (e *EnergyPresetEntity) GetLocationString() string {
	var location string
	if e.City != "" {
		location = e.City
	}
	if e.State != "" {
		if location != "" {
			location += ", " + e.State
		} else {
			location = e.State
		}
	}
	if e.Country != "" {
		if location != "" {
			location += ", " + e.Country
		} else {
			location = e.Country
		}
	}
	return location
}
