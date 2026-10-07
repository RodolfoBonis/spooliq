package services

import "math"

// WasteGramsPerColorChange is the DEFAULT amount of filament (in grams) assumed to
// be purged/wasted on every color change in a multi-color (AMS) print. A print
// with N filaments has N-1 color changes, so the wasted mass is
// WasteGramsPerColorChange * (N-1). It is the fallback used when the item's cost
// preset does not configure a per-color-change waste (see wasteGramsPerChange), so
// pre-existing budgets priced without a preset stay byte-identical.
const WasteGramsPerColorChange = 15.0

// Discount type constants for PricingInput.DiscountType. An empty string means "no
// discount".
const (
	// DiscountTypePercent treats DiscountValue as a percentage (0-100) of base_price.
	DiscountTypePercent = "percent"
	// DiscountTypeFixed treats DiscountValue as a fixed amount in REAIS.
	DiscountTypeFixed = "fixed"
)

// CostPresetInput carries the rates/percentages taken from a cost preset. A nil
// *CostPresetInput means "no preset" (rates default to zero).
//
//   - LaborRatePerHour, PostProcessingCostPerHour, SupportRemovalCostPerHour are in
//     REAIS per hour.
//   - PackagingCostPerItem, QualityControlCostPerItem are in REAIS per product unit.
//   - OverheadPercentage / ProfitMarginPercentage / FailureRatePercentage are plain
//     percentages (e.g. 15 means 15%).
//   - WasteGramsPerColorChange is grams purged per color change; when <= 0 the engine
//     falls back to the package default (WasteGramsPerColorChange constant).
type CostPresetInput struct {
	LaborRatePerHour          float64
	OverheadPercentage        float64
	ProfitMarginPercentage    float64
	PostProcessingCostPerHour float64
	SupportRemovalCostPerHour float64
	PackagingCostPerItem      float64
	QualityControlCostPerItem float64
	FailureRatePercentage     float64
	WasteGramsPerColorChange  float64
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
	PostProcessingMinutes   int
	SupportRemovalMinutes   int
	// CostPreset is the item-level cost preset (its labor/post-processing/support
	// rates, packaging/QC per-item rates, failure rate and waste-per-change drive the
	// item costs; its overhead/profit percentages are only used as the overhead/profit
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

	// IncludeMachineCost toggles the per-item machine-time cost (print hours times
	// MachineCostPerHour). MachineCostPerHour is the machine preset's cost_per_hour in
	// REAIS; it is only charged when IncludeMachineCost is true and the rate is > 0.
	IncludeMachineCost bool
	MachineCostPerHour float64

	// BudgetCostPreset is the budget-level cost preset. When set, its overhead and
	// profit percentages win over any item-level preset.
	BudgetCostPreset *CostPresetInput

	// Discount applied to base_price. DiscountType is "" (none), "percent" (value is a
	// 0-100 percentage of base_price) or "fixed" (value is REAIS). The resulting
	// discount is capped at base_price (never negative net).
	DiscountType  string
	DiscountValue float64

	// Shipping. When IncludeShipping is true the shipping cost is ShippingOverrideCents
	// if set, otherwise round((ShippingCostBase + ShippingCostPerGram*totalGrams)*100)
	// where totalGrams is the summed grams of every item's filaments. The base/per-gram
	// rates come from the budget cost preset and are in REAIS.
	IncludeShipping       bool
	ShippingOverrideCents *int64
	ShippingCostBase      float64
	ShippingCostPerGram   float64

	// TaxRatePercent is the effective tax rate applied "por dentro" (0..<100). The
	// caller resolves it from the budget's tax_rate or the company default before
	// calling. A rate of 0 means no tax.
	TaxRatePercent float64

	Items []PricingItemInput
}

// PricingItemResult is the per-item cost + sale breakdown (all cents).
type PricingItemResult struct {
	FilamentCost       int64
	WasteCost          int64
	EnergyCost         int64
	MachineCost        int64
	SetupCost          int64
	ManualLaborCost    int64
	PostProcessingCost int64
	SupportRemovalCost int64
	PackagingCost      int64
	QualityControlCost int64
	FailureCost        int64
	// ItemTotalCost is the sum of all the direct costs above (no markup).
	ItemTotalCost int64
	// UnitCost is ItemTotalCost / quantity, rounded (cost per unit, no markup).
	UnitCost int64
	// SaleTotal is ItemTotalCost plus this item's proportional share of the
	// budget-wide markup (overhead+profit, less discount, plus tax). The per-item
	// SaleTotal values sum EXACTLY to PricingResult.Total minus shipping, so that the
	// item sale totals PLUS shipping reconcile to the budget total (remainder on the
	// last item).
	SaleTotal int64
	// SaleUnitPrice is SaleTotal / quantity, rounded. Note: SaleUnitPrice * quantity
	// may differ from SaleTotal by a few cents because of rounding.
	SaleUnitPrice int64
}

// PricingResult is the full budget breakdown (all cents unless noted).
type PricingResult struct {
	Items []PricingItemResult

	FilamentCost       int64
	WasteCost          int64
	EnergyCost         int64
	MachineCost        int64
	SetupCost          int64
	LaborCost          int64 // sum of manual labor across items
	PostProcessingCost int64
	SupportRemovalCost int64
	PackagingCost      int64
	QualityControlCost int64
	FailureCost        int64
	Subtotal           int64 // sum of all item direct costs
	Overhead           int64
	Profit             int64
	BasePrice          int64 // Subtotal + Overhead + Profit (sale price before adjustments)

	DiscountAmount int64
	ShippingCost   int64
	TaxAmount      int64
	TaxRateApplied float64 // the effective rate actually applied (percent)

	Total int64 // BasePrice - DiscountAmount + ShippingCost + TaxAmount
}

// wasteGramsPerChange returns the grams purged per color change for an item: the
// item's cost preset value when positive, otherwise the package default. This keeps
// budgets priced without a configured preset byte-identical to the legacy constant.
func wasteGramsPerChange(cp *CostPresetInput) float64 {
	if cp != nil && cp.WasteGramsPerColorChange > 0 {
		return cp.WasteGramsPerColorChange
	}
	return WasteGramsPerColorChange
}

// Calculate computes every budget cost from pure in-memory inputs (no DB access).
//
// Direct per-item costs (cents):
//   - filament = sum over the item's filaments of grams * price/kg;
//   - waste = wasteGramsPerChange(preset) per extra color (N-1 changes), valued at the
//     AVERAGE price/kg of the item's filaments, only when IncludeWasteCost and the item
//     has more than one filament;
//   - energy = power(W) * hours / 1000 * price(R$/kWh), per item, only when EnergyEnabled;
//   - machine = hours * MachineCostPerHour, only when IncludeMachineCost;
//   - setup + manual labor = minutes/60 * the item cost preset's labor rate;
//   - post-processing / support-removal = minutes/60 * the respective preset rate;
//   - packaging = quantity * packaging_cost_per_item; quality control = quantity * qc_cost_per_item;
//   - failure = (filament + waste + energy + machine) * failure_rate_percentage.
//
// Budget:
//   - subtotal = sum of item direct costs; overhead = subtotal * overhead%;
//     profit = (subtotal + overhead) * profit%; base_price = subtotal + overhead + profit;
//   - discount (percent of base or fixed reais), capped at base_price;
//   - shipping (override or base + per-gram*totalGrams) when included;
//   - taxes "por dentro": net = base - discount + shipping; total = net / (1 - rate/100);
//     tax = total - net;
//   - per-item sale values distribute (total - shipping) proportionally to each item's
//     direct cost, with the rounding remainder on the last item, so the item sale totals
//     PLUS shipping reconcile EXACTLY to the total (see DistributeMarkup).
//
// All money is rounded to the nearest cent via the helpers in pricing.go.
func Calculate(in PricingInput) (PricingResult, error) {
	var result PricingResult
	result.Items = make([]PricingItemResult, len(in.Items))

	itemCosts := make([]int64, len(in.Items))
	var totalGrams float64

	for i, item := range in.Items {
		var filamentCost int64
		var priceSum float64
		for _, f := range item.Filaments {
			filamentCost += FilamentCostCents(f.Grams, f.PricePerKgCents)
			priceSum += f.PricePerKgCents
			totalGrams += f.Grams
		}

		var wasteCost int64
		if in.IncludeWasteCost && len(item.Filaments) > 1 {
			avgPrice := priceSum / float64(len(item.Filaments))
			wasteGrams := wasteGramsPerChange(item.CostPreset) * float64(len(item.Filaments)-1)
			wasteCost = WasteCostCents(wasteGrams, avgPrice)
		}

		hours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0

		var energyCost int64
		if in.EnergyEnabled {
			energyCost = EnergyCostCents(in.MachinePowerWatts, hours, in.EnergyPricePerKwh)
		}

		var machineCost int64
		if in.IncludeMachineCost {
			machineCost = HourlyCostCents(hours, in.MachineCostPerHour)
		}

		var laborRate, postRate, supportRate, packagingPerItem, qcPerItem, failurePct float64
		if item.CostPreset != nil {
			laborRate = item.CostPreset.LaborRatePerHour
			postRate = item.CostPreset.PostProcessingCostPerHour
			supportRate = item.CostPreset.SupportRemovalCostPerHour
			packagingPerItem = item.CostPreset.PackagingCostPerItem
			qcPerItem = item.CostPreset.QualityControlCostPerItem
			failurePct = item.CostPreset.FailureRatePercentage
		}

		setupCost := LaborCostCents(item.SetupTimeMinutes, laborRate)
		manualLaborCost := LaborCostCents(item.ManualLaborMinutesTotal, laborRate)
		postProcessingCost := LaborCostCents(item.PostProcessingMinutes, postRate)
		supportRemovalCost := LaborCostCents(item.SupportRemovalMinutes, supportRate)
		packagingCost := ReaisPerItemCents(item.Quantity, packagingPerItem)
		qualityControlCost := ReaisPerItemCents(item.Quantity, qcPerItem)
		failureCost := PercentageCents(filamentCost+wasteCost+energyCost+machineCost, failurePct)

		itemTotal := filamentCost + wasteCost + energyCost + machineCost + setupCost + manualLaborCost +
			postProcessingCost + supportRemovalCost + packagingCost + qualityControlCost + failureCost

		result.Items[i] = PricingItemResult{
			FilamentCost:       filamentCost,
			WasteCost:          wasteCost,
			EnergyCost:         energyCost,
			MachineCost:        machineCost,
			SetupCost:          setupCost,
			ManualLaborCost:    manualLaborCost,
			PostProcessingCost: postProcessingCost,
			SupportRemovalCost: supportRemovalCost,
			PackagingCost:      packagingCost,
			QualityControlCost: qualityControlCost,
			FailureCost:        failureCost,
			ItemTotalCost:      itemTotal,
			UnitCost:           UnitPriceCents(itemTotal, item.Quantity),
		}
		itemCosts[i] = itemTotal

		result.FilamentCost += filamentCost
		result.WasteCost += wasteCost
		result.EnergyCost += energyCost
		result.MachineCost += machineCost
		result.SetupCost += setupCost
		result.LaborCost += manualLaborCost
		result.PostProcessingCost += postProcessingCost
		result.SupportRemovalCost += supportRemovalCost
		result.PackagingCost += packagingCost
		result.QualityControlCost += qualityControlCost
		result.FailureCost += failureCost
		result.Subtotal += itemTotal
	}

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
	result.BasePrice = result.Subtotal + result.Overhead + result.Profit

	// Discount on base_price, capped so net never goes negative.
	var discount int64
	switch in.DiscountType {
	case DiscountTypePercent:
		discount = PercentageCents(result.BasePrice, in.DiscountValue)
	case DiscountTypeFixed:
		discount = toCents(in.DiscountValue)
	}
	if discount < 0 {
		discount = 0
	}
	if discount > result.BasePrice {
		discount = result.BasePrice
	}
	result.DiscountAmount = discount

	// Shipping (override or computed from the budget cost preset rates).
	var shipping int64
	if in.IncludeShipping {
		switch {
		case in.ShippingOverrideCents != nil:
			shipping = *in.ShippingOverrideCents
		default:
			shipping = int64(math.Round((in.ShippingCostBase + in.ShippingCostPerGram*totalGrams) * 100.0))
		}
		if shipping < 0 {
			shipping = 0
		}
	}
	result.ShippingCost = shipping

	// Taxes "por dentro": gross the net up so the tax is included in the total.
	net := result.BasePrice - discount + shipping
	result.TaxRateApplied = in.TaxRatePercent
	total := TaxInsideTotalCents(net, in.TaxRatePercent)
	result.TaxAmount = total - net
	result.Total = total

	// Distribute the markup across items so the per-item sale totals PLUS shipping
	// reconcile EXACTLY to the total.
	saleTotals := DistributeMarkup(itemCosts, result.Total-result.ShippingCost)
	for i := range result.Items {
		result.Items[i].SaleTotal = saleTotals[i]
		result.Items[i].SaleUnitPrice = UnitPriceCents(saleTotals[i], in.Items[i].Quantity)
	}

	return result, nil
}

// DistributeMarkup splits a budget's markup across items in proportion to each
// item's direct cost (itemCosts), returning the final value (direct cost + markup
// share) in cents for each item.
//
// The markup to distribute is derived as targetTotal - sum(itemCosts); allocating it
// this way and assigning any rounding remainder to the last item guarantees the
// returned slice sums EXACTLY to targetTotal, so per-item subtotals always reconcile
// with the target line. math.Round is used (never truncation) so shares are the
// nearest cent. If the items have no direct cost to weight by (sum <= 0) the whole
// markup is placed on the last item.
//
// This is the single source of truth for per-item sale distribution, shared by the
// pricing engine, the API responses and the PDF. Callers that charge shipping pass a
// target of (total - shipping) so the item sale totals plus shipping equal the total.
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
