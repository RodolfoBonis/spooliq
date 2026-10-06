// Package services contains pure, side-effect-free business calculations for budgets.
//
// IMPORTANT domain invariants:
//   - Filament price (price_per_kg) is stored in CENTS per kg (e.g. 12000 == R$120,00/kg).
//   - Preset rates (energy per kWh, labor per hour) are stored in REAIS.
//   - All budget/item costs are stored as int64 CENTS (e.g. 1200 == R$12,00).
//
// Every helper here converts to cents and rounds (never truncates) so repeated
// calculations stay accurate to the cent.
package services

import "math"

// toCents rounds a reais amount to an integer number of cents.
func toCents(reais float64) int64 {
	return int64(math.Round(reais * 100.0))
}

// FilamentCostCents computes the cost in cents of using `grams` of a filament
// priced at `pricePerKgCents` cents per kilogram.
//
// Example: 100g at 12000 cents/kg (R$120/kg) => 100/1000 * 12000 = 1200 cents.
func FilamentCostCents(grams, pricePerKgCents float64) int64 {
	if grams <= 0 || pricePerKgCents <= 0 {
		return 0
	}
	return int64(math.Round((grams / 1000.0) * pricePerKgCents))
}

// WasteCostCents computes the cost in cents of `wasteGrams` of wasted filament
// priced at `pricePerKgCents` cents per kilogram. Uses the same formula as
// FilamentCostCents but kept as a named function for call-site clarity.
func WasteCostCents(wasteGrams, pricePerKgCents float64) int64 {
	return FilamentCostCents(wasteGrams, pricePerKgCents)
}

// EnergyCostCents computes energy cost in cents given machine power draw in watts,
// print time in hours and the energy price per kWh (in reais).
func EnergyCostCents(powerWatts, hours, energyPricePerKwhReais float64) int64 {
	if powerWatts <= 0 || hours <= 0 || energyPricePerKwhReais <= 0 {
		return 0
	}
	kwh := powerWatts * hours / 1000.0
	return toCents(kwh * energyPricePerKwhReais)
}

// LaborCostCents converts `minutes` of labor at `ratePerHourReais` into cents.
// Used for both setup time and manual labor time.
func LaborCostCents(minutes int, ratePerHourReais float64) int64 {
	if minutes <= 0 || ratePerHourReais <= 0 {
		return 0
	}
	hours := float64(minutes) / 60.0
	return toCents(hours * ratePerHourReais)
}

// PercentageCents applies `percentage` (e.g. 15 for 15%) to a cents base and
// returns the resulting amount in cents. Used for overhead and profit.
func PercentageCents(baseCents int64, percentage float64) int64 {
	if percentage <= 0 || baseCents == 0 {
		return 0
	}
	return int64(math.Round(float64(baseCents) * percentage / 100.0))
}

// UnitPriceCents divides a total cents amount by a product quantity, rounding to
// the nearest cent. Returns 0 when quantity is not positive.
func UnitPriceCents(totalCents int64, quantity int) int64 {
	if quantity <= 0 {
		return 0
	}
	return int64(math.Round(float64(totalCents) / float64(quantity)))
}
