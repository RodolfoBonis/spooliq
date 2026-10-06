package services

import "testing"

// TestCalculate_Components drives every cost component through Calculate and the
// documented edge cases: no items, missing presets, multi-color waste, the
// quantity-0 guard, rounding and the exact-sum invariant between per-item sale
// values and the budget total.
func TestCalculate_Components(t *testing.T) {
	// A standard cost preset: R$60/h labor, 10% overhead, 20% profit.
	preset := &CostPresetInput{LaborRatePerHour: 60, OverheadPercentage: 10, ProfitMarginPercentage: 20}

	tests := []struct {
		name string
		in   PricingInput
		// expectations on the result
		wantItems    []PricingItemResult
		wantFilament int64
		wantWaste    int64
		wantEnergy   int64
		wantSetup    int64
		wantLabor    int64
		wantSubtotal int64
		wantOverhead int64
		wantProfit   int64
		wantTotal    int64
	}{
		{
			name: "empty budget: zero everything",
			in:   PricingInput{},
		},
		{
			name: "single item, filament only, no presets",
			in: PricingInput{
				Items: []PricingItemInput{
					{
						Quantity:  1,
						Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}, // 1200
					},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1200, SaleUnitPrice: 1200},
			},
			wantFilament: 1200,
			wantSubtotal: 1200,
			wantTotal:    1200,
		},
		{
			name: "multi-color waste at average price (2 colors => 15g at avg)",
			in: PricingInput{
				IncludeWasteCost: true,
				Items: []PricingItemInput{
					{
						Quantity: 1,
						Filaments: []PricingFilamentInput{
							{Grams: 100, PricePerKgCents: 12000}, // 1200
							{Grams: 50, PricePerKgCents: 20000},  // 1000
						},
					},
				},
			},
			// avg price = (12000+20000)/2 = 16000; waste 15g => round(15/1000*16000)=240
			wantItems: []PricingItemResult{
				{FilamentCost: 2200, WasteCost: 240, ItemTotalCost: 2440, UnitCost: 2440, SaleTotal: 2440, SaleUnitPrice: 2440},
			},
			wantFilament: 2200,
			wantWaste:    240,
			wantSubtotal: 2440,
			wantTotal:    2440,
		},
		{
			name: "waste ignored when flag off even with multiple colors",
			in: PricingInput{
				IncludeWasteCost: false,
				Items: []PricingItemInput{
					{
						Quantity: 1,
						Filaments: []PricingFilamentInput{
							{Grams: 100, PricePerKgCents: 12000},
							{Grams: 50, PricePerKgCents: 20000},
						},
					},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 2200, ItemTotalCost: 2200, UnitCost: 2200, SaleTotal: 2200, SaleUnitPrice: 2200},
			},
			wantFilament: 2200,
			wantSubtotal: 2200,
			wantTotal:    2200,
		},
		{
			name: "waste ignored for single color even when flag on",
			in: PricingInput{
				IncludeWasteCost: true,
				Items: []PricingItemInput{
					{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1200, SaleUnitPrice: 1200},
			},
			wantFilament: 1200,
			wantSubtotal: 1200,
			wantTotal:    1200,
		},
		{
			name: "energy when enabled (200W, 10h, R$1/kWh => 200)",
			in: PricingInput{
				EnergyEnabled:     true,
				MachinePowerWatts: 200,
				EnergyPricePerKwh: 1.00,
				Items: []PricingItemInput{
					{Quantity: 1, PrintTimeHours: 10, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, EnergyCost: 200, ItemTotalCost: 1400, UnitCost: 1400, SaleTotal: 1400, SaleUnitPrice: 1400},
			},
			wantFilament: 1200,
			wantEnergy:   200,
			wantSubtotal: 1400,
			wantTotal:    1400,
		},
		{
			name: "energy ignored when not enabled",
			in: PricingInput{
				EnergyEnabled:     false,
				MachinePowerWatts: 200,
				EnergyPricePerKwh: 1.00,
				Items: []PricingItemInput{
					{Quantity: 1, PrintTimeHours: 10, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1200, SaleUnitPrice: 1200},
			},
			wantFilament: 1200,
			wantSubtotal: 1200,
			wantTotal:    1200,
		},
		{
			name: "setup + manual labor from item cost preset",
			in: PricingInput{
				Items: []PricingItemInput{
					{
						Quantity:                1,
						SetupTimeMinutes:        30, // R$30 => 3000
						ManualLaborMinutesTotal: 90, // R$90 => 9000
						CostPreset:              &CostPresetInput{LaborRatePerHour: 60},
						Filaments:               []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}},
					},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, SetupCost: 3000, ManualLaborCost: 9000, ItemTotalCost: 13200, UnitCost: 13200, SaleTotal: 13200, SaleUnitPrice: 13200},
			},
			wantFilament: 1200,
			wantSetup:    3000,
			wantLabor:    9000,
			wantSubtotal: 13200,
			wantTotal:    13200,
		},
		{
			name: "labor is zero when item has no cost preset",
			in: PricingInput{
				Items: []PricingItemInput{
					{
						Quantity:                1,
						SetupTimeMinutes:        30,
						ManualLaborMinutesTotal: 90,
						Filaments:               []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}},
					},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1200, SaleUnitPrice: 1200},
			},
			wantFilament: 1200,
			wantSubtotal: 1200,
			wantTotal:    1200,
		},
		{
			name: "overhead and profit from budget cost preset",
			in: PricingInput{
				BudgetCostPreset: preset,
				Items: []PricingItemInput{
					{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}}, // 10000
				},
			},
			// subtotal 10000; overhead 10% => 1000; profit 20% of 11000 => 2200; total 13200
			wantItems: []PricingItemResult{
				{FilamentCost: 10000, ItemTotalCost: 10000, UnitCost: 10000, SaleTotal: 13200, SaleUnitPrice: 13200},
			},
			wantFilament: 10000,
			wantSubtotal: 10000,
			wantOverhead: 1000,
			wantProfit:   2200,
			wantTotal:    13200,
		},
		{
			name: "overhead and profit fall back to first item's preset",
			in: PricingInput{
				Items: []PricingItemInput{
					{Quantity: 1, CostPreset: preset, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 10000, ItemTotalCost: 10000, UnitCost: 10000, SaleTotal: 13200, SaleUnitPrice: 13200},
			},
			wantFilament: 10000,
			wantSubtotal: 10000,
			wantOverhead: 1000,
			wantProfit:   2200,
			wantTotal:    13200,
		},
		{
			name: "quantity > 1: unit cost and unit sale price are per-unit rounded",
			in: PricingInput{
				BudgetCostPreset: &CostPresetInput{OverheadPercentage: 0, ProfitMarginPercentage: 50},
				Items: []PricingItemInput{
					{Quantity: 3, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}}, // 10000
				},
			},
			// subtotal 10000; profit 50% => 5000; total 15000
			// unit cost = round(10000/3) = 3333; sale unit = round(15000/3) = 5000
			wantItems: []PricingItemResult{
				{FilamentCost: 10000, ItemTotalCost: 10000, UnitCost: 3333, SaleTotal: 15000, SaleUnitPrice: 5000},
			},
			wantFilament: 10000,
			wantSubtotal: 10000,
			wantProfit:   5000,
			wantTotal:    15000,
		},
		{
			name: "quantity 0 guard: unit values are 0, totals still computed",
			in: PricingInput{
				Items: []PricingItemInput{
					{Quantity: 0, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}},
				},
			},
			wantItems: []PricingItemResult{
				{FilamentCost: 10000, ItemTotalCost: 10000, UnitCost: 0, SaleTotal: 10000, SaleUnitPrice: 0},
			},
			wantFilament: 10000,
			wantSubtotal: 10000,
			wantTotal:    10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got.Items) != len(tt.wantItems) {
				t.Fatalf("item count: got %d want %d", len(got.Items), len(tt.wantItems))
			}
			for i, w := range tt.wantItems {
				if got.Items[i] != w {
					t.Errorf("item[%d] = %+v, want %+v", i, got.Items[i], w)
				}
			}

			if got.FilamentCost != tt.wantFilament {
				t.Errorf("FilamentCost = %d, want %d", got.FilamentCost, tt.wantFilament)
			}
			if got.WasteCost != tt.wantWaste {
				t.Errorf("WasteCost = %d, want %d", got.WasteCost, tt.wantWaste)
			}
			if got.EnergyCost != tt.wantEnergy {
				t.Errorf("EnergyCost = %d, want %d", got.EnergyCost, tt.wantEnergy)
			}
			if got.SetupCost != tt.wantSetup {
				t.Errorf("SetupCost = %d, want %d", got.SetupCost, tt.wantSetup)
			}
			if got.LaborCost != tt.wantLabor {
				t.Errorf("LaborCost = %d, want %d", got.LaborCost, tt.wantLabor)
			}
			if got.Subtotal != tt.wantSubtotal {
				t.Errorf("Subtotal = %d, want %d", got.Subtotal, tt.wantSubtotal)
			}
			if got.Overhead != tt.wantOverhead {
				t.Errorf("Overhead = %d, want %d", got.Overhead, tt.wantOverhead)
			}
			if got.Profit != tt.wantProfit {
				t.Errorf("Profit = %d, want %d", got.Profit, tt.wantProfit)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}

			// Exact-sum invariant: per-item SaleTotal must reconcile with the budget total.
			var saleSum int64
			for _, it := range got.Items {
				saleSum += it.SaleTotal
			}
			if len(got.Items) > 0 && saleSum != got.Total {
				t.Errorf("sum(SaleTotal) = %d, want Total %d", saleSum, got.Total)
			}
		})
	}
}

// TestCalculate_MultiItemSaleDistribution proves the markup is spread across
// multiple items proportionally to their direct cost and reconciles exactly,
// including when rounding forces a remainder onto the last item.
func TestCalculate_MultiItemSaleDistribution(t *testing.T) {
	in := PricingInput{
		BudgetCostPreset: &CostPresetInput{ProfitMarginPercentage: 10}, // 10% profit, no overhead
		Items: []PricingItemInput{
			{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}}, // 10000
			{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 20000}}}, // 20000
			{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 30000}}}, // 30000
		},
	}

	got, err := Calculate(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// subtotal 60000; profit 10% => 6000; total 66000
	if got.Total != 66000 {
		t.Fatalf("Total = %d, want 66000", got.Total)
	}

	// markup 6000 distributed 10000/20000/30000 => 1000/2000/3000
	wantSale := []int64{11000, 22000, 33000}
	var sum int64
	for i, it := range got.Items {
		if it.SaleTotal != wantSale[i] {
			t.Errorf("item[%d].SaleTotal = %d, want %d", i, it.SaleTotal, wantSale[i])
		}
		sum += it.SaleTotal
	}
	if sum != got.Total {
		t.Errorf("sum(SaleTotal) = %d, want %d", sum, got.Total)
	}
}

// TestDistributeMarkup verifies that the per-item final costs always sum EXACTLY
// to the budget total, that the rounding remainder lands on the last item, and
// that the degenerate cases (no items, zero direct cost) are handled.
func TestDistributeMarkup(t *testing.T) {
	tests := []struct {
		name      string
		itemCosts []int64
		totalCost int64
		want      []int64
	}{
		{name: "no items", itemCosts: nil, totalCost: 0, want: []int64{}},
		{name: "single item absorbs all markup", itemCosts: []int64{1000}, totalCost: 1500, want: []int64{1500}},
		{name: "no markup leaves costs untouched", itemCosts: []int64{1000, 2000, 3000}, totalCost: 6000, want: []int64{1000, 2000, 3000}},
		{name: "even split", itemCosts: []int64{1000, 1000}, totalCost: 3000, want: []int64{1500, 1500}},
		{name: "remainder lands on last item", itemCosts: []int64{1, 1, 1}, totalCost: 13, want: []int64{4, 4, 5}},
		{name: "proportional to direct cost", itemCosts: []int64{100, 300}, totalCost: 800, want: []int64{200, 600}},
		{name: "zero direct cost puts markup on last item", itemCosts: []int64{0, 0}, totalCost: 500, want: []int64{0, 500}},
		{name: "negative markup still reconciles", itemCosts: []int64{1000, 1000}, totalCost: 1500, want: []int64{750, 750}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistributeMarkup(tt.itemCosts, tt.totalCost)
			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %v want %v", got, tt.want)
			}
			var sum int64
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %d want %d (full: %v)", i, got[i], tt.want[i], got)
				}
				sum += got[i]
			}
			if len(got) > 0 && sum != tt.totalCost {
				t.Errorf("sum %d != totalCost %d", sum, tt.totalCost)
			}
		})
	}
}

// TestDistributeMarkupAlwaysSumsToTotal is a property-style check over a range of
// uneven item costs ensuring the exact-sum invariant holds regardless of rounding.
func TestDistributeMarkupAlwaysSumsToTotal(t *testing.T) {
	cases := [][]int64{
		{7, 11, 13, 17},
		{1, 2, 3, 4, 5, 6, 7},
		{999, 1, 1},
		{333, 333, 334},
	}
	markups := []int64{0, 1, 7, 100, 9999, -50}

	for _, costs := range cases {
		var subtotal int64
		for _, c := range costs {
			subtotal += c
		}
		for _, m := range markups {
			total := subtotal + m
			got := DistributeMarkup(costs, total)
			var sum int64
			for _, v := range got {
				sum += v
			}
			if sum != total {
				t.Errorf("costs=%v total=%d: sum=%d (got %v)", costs, total, sum, got)
			}
		}
	}
}
