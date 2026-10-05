package services

import "math"

// WasteGramsPerColorChange is the fixed amount of filament (in grams) assumed to
// be purged/wasted on every color change in a multi-color (AMS) print. A print
// with N filaments has N-1 color changes, so the wasted mass is
// WasteGramsPerColorChange * (N-1). This mirrors the long-standing production
// behaviour and is kept as a named constant so there is a single place to tune it.
const WasteGramsPerColorChange = 15.0

// CostPresetInput carries the rates/percentages taken from a cost preset. A nil
// *CostPresetInput means "no preset" (rates default to zero).
//
//   - LaborRatePerHour is in REAIS per hour (used for setup + manual labor).
//   - OverheadPercentage / ProfitMarginPercentage are plain percentages (e.g. 15
//     means 15%).
type CostPresetInput struct {
	LaborRatePerHour       float64
	OverheadPercentage     float64
	ProfitMarginPercentage float64
}

// PricingFilamentInput is one filament used by an item.
//
//   - Grams is the TOTAL grams of this filament for the item.
//   - PricePerKgCents is the filament price in CENTS per kilogram.
type PricingFilamentInput struct {
	Grams           float64
	PricePerKgCents float64
}

// PricingItemInput is a single product line in the budget.
type PricingItemInput struct {
	Quantity                int // product units
	PrintTimeHours          int
	PrintTimeMinutes        int
	SetupTimeMinutes        int
	ManualLaborMinutesTotal int
	// CostPreset is the item-level cost preset (its labor rate drives setup +
	// manual labor cost; its percentages are only used as the overhead/profit
	// fallback when the item is the first item and no budget-level preset is set).
	CostPreset *CostPresetInput
	Filaments  []PricingFilamentInput
}

// PricingInput is the complete, DB-free input to Calculate.
type PricingInput struct {
	IncludeWasteCost bool
	// EnergyEnabled is the resolved decision "charge energy": the caller sets it to
	// IncludeEnergyCost AND both the machine and energy presets being present.
	EnergyEnabled     bool
	MachinePowerWatts float64 // machine power draw in watts
	EnergyPricePerKwh float64 // energy price in REAIS per kWh
	// BudgetCostPreset is the budget-level cost preset. When set, its overhead and
	// profit percentages win over any item-level preset.
	BudgetCostPreset *CostPresetInput
	Items            []PricingItemInput
}

// PricingItemResult is the per-item cost + sale breakdown (all cents).
type PricingItemResult struct {
	FilamentCost    int64
	WasteCost       int64
	EnergyCost      int64
	SetupCost       int64
	ManualLaborCost int64
	// ItemTotalCost is the sum of the five direct costs above (no markup).
	ItemTotalCost int64
	// UnitCost is ItemTotalCost / quantity, rounded (cost per unit, no markup).
	UnitCost int64
	// SaleTotal is ItemTotalCost plus this item's proportional share of the
	// budget-wide overhead+profit markup. The per-item SaleTotal values sum EXACTLY
	// to PricingResult.Total (remainder on the last item).
	SaleTotal int64
	// SaleUnitPrice is SaleTotal / quantity, rounded. Note: SaleUnitPrice * quantity
	// may differ from SaleTotal by a few cents because of rounding.
	SaleUnitPrice int64
}

// PricingResult is the full budget breakdown (all cents).
type PricingResult struct {
	Items []PricingItemResult

	FilamentCost int64
	WasteCost    int64
	EnergyCost   int64
	SetupCost    int64
	LaborCost    int64 // sum of manual labor across items
	Subtotal     int64 // Filament + Waste + Energy + Setup + Labor
	Overhead     int64
	Profit       int64
	Total        int64 // Subtotal + Overhead + Profit
}

// Calculate computes every budget cost from pure in-memory inputs (no DB access),
// preserving the production semantics:
//
//   - filament cost = sum over the item's filaments of grams * price/kg;
//   - waste = WasteGramsPerColorChange per extra color (N-1 changes), valued at the
//     AVERAGE price/kg of the item's filaments, only when IncludeWasteCost is set
//     and the item has more than one filament;
//   - energy = power(W) * hours / 1000 * price(R$/kWh), per item, only when
//     EnergyEnabled;
//   - setup + manual labor = minutes/60 * the item cost preset's labor rate;
//   - overhead = subtotal * overhead%, profit = (subtotal + overhead) * profit%,
//     where the percentages come from the budget cost preset, falling back to the
//     FIRST item's cost preset;
//   - per-item sale values distribute overhead+profit proportionally to each item's
//     direct cost, with the rounding remainder on the last item so the shares sum
//     EXACTLY to the total (see DistributeMarkup).
//
// All money is rounded to the nearest cent via the helpers in pricing.go.
func Calculate(in PricingInput) (PricingResult, error) {
	var result PricingResult
	result.Items = make([]PricingItemResult, len(in.Items))

	itemCosts := make([]int64, len(in.Items))

	for i, item := range in.Items {
		var filamentCost int64
		var priceSum float64
		for _, f := range item.Filaments {
			filamentCost += FilamentCostCents(f.Grams, f.PricePerKgCents)
			priceSum += f.PricePerKgCents
		}

		var wasteCost int64
		if in.IncludeWasteCost && len(item.Filaments) > 1 {
			avgPrice := priceSum / float64(len(item.Filaments))
			wasteGrams := WasteGramsPerColorChange * float64(len(item.Filaments)-1)
			wasteCost = WasteCostCents(wasteGrams, avgPrice)
		}

		var energyCost int64
		if in.EnergyEnabled {
			hours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0
			energyCost = EnergyCostCents(in.MachinePowerWatts, hours, in.EnergyPricePerKwh)
		}

		var laborRate float64
		if item.CostPreset != nil {
			laborRate = item.CostPreset.LaborRatePerHour
		}
		setupCost := LaborCostCents(item.SetupTimeMinutes, laborRate)
		manualLaborCost := LaborCostCents(item.ManualLaborMinutesTotal, laborRate)

		itemTotal := filamentCost + wasteCost + energyCost + setupCost + manualLaborCost

		result.Items[i] = PricingItemResult{
			FilamentCost:    filamentCost,
			WasteCost:       wasteCost,
			EnergyCost:      energyCost,
			SetupCost:       setupCost,
			ManualLaborCost: manualLaborCost,
			ItemTotalCost:   itemTotal,
			UnitCost:        UnitPriceCents(itemTotal, item.Quantity),
		}
		itemCosts[i] = itemTotal

		result.FilamentCost += filamentCost
		result.WasteCost += wasteCost
		result.EnergyCost += energyCost
		result.SetupCost += setupCost
		result.LaborCost += manualLaborCost
	}

	result.Subtotal = result.FilamentCost + result.WasteCost + result.EnergyCost + result.SetupCost + result.LaborCost

	// Overhead/profit percentages: budget-level preset wins, else the first item's.
	var overheadPct, profitPct float64
	if in.BudgetCostPreset != nil {
		overheadPct = in.BudgetCostPreset.OverheadPercentage
		profitPct = in.BudgetCostPreset.ProfitMarginPercentage
	} else if len(in.Items) > 0 && in.Items[0].CostPreset != nil {
		overheadPct = in.Items[0].CostPreset.OverheadPercentage
		profitPct = in.Items[0].CostPreset.ProfitMarginPercentage
	}

	result.Overhead = PercentageCents(result.Subtotal, overheadPct)
	result.Profit = PercentageCents(result.Subtotal+result.Overhead, profitPct)
	result.Total = result.Subtotal + result.Overhead + result.Profit

	// Distribute the overhead+profit markup across items proportionally to their
	// direct cost; shares sum EXACTLY to result.Total.
	saleTotals := DistributeMarkup(itemCosts, result.Total)
	for i := range result.Items {
		result.Items[i].SaleTotal = saleTotals[i]
		result.Items[i].SaleUnitPrice = UnitPriceCents(saleTotals[i], in.Items[i].Quantity)
	}

	return result, nil
}

// DistributeMarkup splits a budget's overhead+profit markup across items in
// proportion to each item's direct cost (itemCosts), returning the final cost
// (direct cost + markup share) in cents for each item.
//
// The markup to distribute is derived as totalCost - sum(itemCosts); allocating it
// this way and assigning any rounding remainder to the last item guarantees the
// returned slice sums EXACTLY to totalCost, so per-item subtotals always reconcile
// with the TOTAL line. math.Round is used (never truncation) so shares are the
// nearest cent. If the items have no direct cost to weight by (sum <= 0) the whole
// markup is placed on the last item.
//
// This is the single source of truth for per-item sale distribution, shared by the
// pricing engine, the API responses and the PDF.
func DistributeMarkup(itemCosts []int64, totalCost int64) []int64 {
	n := len(itemCosts)
	result := make([]int64, n)
	if n == 0 {
		return result
	}

	var subtotal int64
	for _, c := range itemCosts {
		subtotal += c
	}
	markup := totalCost - subtotal

	if subtotal <= 0 {
		copy(result, itemCosts)
		result[n-1] += markup
		return result
	}

	var allocated int64
	for i := 0; i < n; i++ {
		var share int64
		if i == n-1 {
			// Last item absorbs the remainder so the total is exact.
			share = markup - allocated
		} else {
			share = int64(math.Round(float64(markup) * float64(itemCosts[i]) / float64(subtotal)))
			allocated += share
		}
		result[i] = itemCosts[i] + share
	}
	return result
}
