package entities

import (
	"strconv"
	"strings"
)

// Fallback names used when there is not enough information to build a
// descriptive name for an auto-generated preset.
const (
	// FallbackMachineName is used when a machine preset has no brand/model/nozzle.
	FallbackMachineName = "Máquina sem nome"
	// FallbackEnergyName is used when an energy preset has no location/provider/tariff.
	FallbackEnergyName = "Tarifa de energia"
	// FallbackCostName is used when a cost preset has no labor/margin information.
	FallbackCostName = "Perfil de custos"
)

// formatDecimal formats a float using a dot as the decimal separator, trimming
// insignificant trailing zeros (e.g. 0.40 -> "0.4", 150 -> "150").
func formatDecimal(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}

// formatDecimalComma is like formatDecimal but uses a comma as the decimal
// separator, matching Brazilian (pt-BR) number formatting (e.g. 0.75 -> "0,75").
func formatDecimalComma(value float32) string {
	return strings.Replace(formatDecimal(value), ".", ",", 1)
}

// joinNonEmpty joins the non-empty parts with sep, gracefully dropping empties.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, strings.TrimSpace(p))
		}
	}
	return strings.Join(kept, sep)
}

// GenerateMachineName builds a human-friendly pt-BR name for a machine preset
// from its brand, model and nozzle diameter (in mm). Empty brand/model parts are
// dropped; the nozzle is only included when greater than zero. When nothing is
// available it returns FallbackMachineName.
//
// Example: ("Bambu Lab", "P1S", 0.4) -> "Bambu Lab P1S · bico 0.4mm".
func GenerateMachineName(brand, model string, nozzleDiameter float32) string {
	identity := joinNonEmpty(" ", brand, model)

	var nozzle string
	if nozzleDiameter > 0 {
		nozzle = "bico " + formatDecimal(nozzleDiameter) + "mm"
	}

	name := joinNonEmpty(" · ", identity, nozzle)
	if name == "" {
		return FallbackMachineName
	}
	return name
}

// GenerateEnergyName builds a human-friendly pt-BR name for an energy preset.
// The location head prefers the provider, falling back to the city; the state is
// appended as "/{State}" when present. The tariff (R$ x,yz/kWh) is included when
// greater than zero. When nothing is available it returns FallbackEnergyName.
//
// Example: ("CEMIG", "", "MG", 0.95) -> "CEMIG/MG · R$ 0,95/kWh".
func GenerateEnergyName(provider, city, state string, energyCostPerKwh float32) string {
	head := provider
	if strings.TrimSpace(head) == "" {
		head = city
	}

	location := strings.TrimSpace(head)
	if trimmedState := strings.TrimSpace(state); trimmedState != "" {
		if location != "" {
			location += "/" + trimmedState
		} else {
			location = trimmedState
		}
	}

	var tariff string
	if energyCostPerKwh > 0 {
		tariff = "R$ " + formatDecimalComma(energyCostPerKwh) + "/kWh"
	}

	name := joinNonEmpty(" · ", location, tariff)
	if name == "" {
		return FallbackEnergyName
	}
	return name
}

// GenerateCostName builds a human-friendly pt-BR name for a cost preset from its
// labor rate (R$/h) and profit margin (%). Parts are dropped when not greater
// than zero. When nothing is available it returns FallbackCostName.
//
// Example: (50, 20) -> "Mão de obra R$ 50/h · margem 20%".
func GenerateCostName(laborCostPerHour, profitMarginPercentage float32) string {
	var labor string
	if laborCostPerHour > 0 {
		labor = "Mão de obra R$ " + formatDecimalComma(laborCostPerHour) + "/h"
	}

	var margin string
	if profitMarginPercentage > 0 {
		margin = "margem " + formatDecimal(profitMarginPercentage) + "%"
	}

	name := joinNonEmpty(" · ", labor, margin)
	if name == "" {
		return FallbackCostName
	}
	return name
}
