package services

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// TestCalculate_NewComponents drives each new direct-cost component in isolation so
// the per-item and budget breakdowns are pinned exactly.
func TestCalculate_NewComponents(t *testing.T) {
	int64p := func(v int64) *int64 { return &v }
	_ = int64p

	tests := []struct {
		name          string
		in            PricingInput
		wantItem      PricingItemResult
		wantSubtotal  int64
		wantBasePrice int64
		wantTotal     int64
	}{
		{
			name: "machine cost = hours * cost_per_hour",
			in: PricingInput{
				IncludeMachineCost: true,
				MachineCostPerHour: 10, // R$10/h
				Items: []PricingItemInput{
					{Quantity: 1, PrintTimeHours: 2, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}}, // 1200
				},
			},
			// machine = 2h * R$10 = R$20 = 2000
			wantItem:      PricingItemResult{FilamentCost: 1200, MachineCost: 2000, ItemTotalCost: 3200, UnitCost: 3200, SaleTotal: 3200, SaleUnitPrice: 3200},
			wantSubtotal:  3200,
			wantBasePrice: 3200,
			wantTotal:     3200,
		},
		{
			name: "machine cost ignored when flag off",
			in: PricingInput{
				IncludeMachineCost: false,
				MachineCostPerHour: 10,
				Items: []PricingItemInput{
					{Quantity: 1, PrintTimeHours: 2, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}},
				},
			},
			wantItem:      PricingItemResult{FilamentCost: 1200, ItemTotalCost: 1200, UnitCost: 1200, SaleTotal: 1200, SaleUnitPrice: 1200},
			wantSubtotal:  1200,
			wantBasePrice: 1200,
			wantTotal:     1200,
		},
		{
			name: "post-processing + support removal from item preset",
			in: PricingInput{
				Items: []PricingItemInput{
					{
						Quantity:              1,
						PostProcessingMinutes: 40, // 40/60 * R$30 = R$20 = 2000
						SupportRemovalMinutes: 30, // 30/60 * R$60 = R$30 = 3000
						CostPreset:            &CostPresetInput{PostProcessingCostPerHour: 30, SupportRemovalCostPerHour: 60},
						Filaments:             []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}},
					},
				},
			},
			wantItem:      PricingItemResult{FilamentCost: 1200, PostProcessingCost: 2000, SupportRemovalCost: 3000, ItemTotalCost: 6200, UnitCost: 6200, SaleTotal: 6200, SaleUnitPrice: 6200},
			wantSubtotal:  6200,
			wantBasePrice: 6200,
			wantTotal:     6200,
		},
		{
			name: "packaging + quality control scale with product quantity",
			in: PricingInput{
				Items: []PricingItemInput{
					{
						Quantity:   3,
						CostPreset: &CostPresetInput{PackagingCostPerItem: 2, QualityControlCostPerItem: 1}, // 3*2=600, 3*1=300
						Filaments:  []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}},
					},
				},
			},
			// filament 1200; packaging 600; qc 300; total 2100; unit = round(2100/3)=700
			wantItem:      PricingItemResult{FilamentCost: 1200, PackagingCost: 600, QualityControlCost: 300, ItemTotalCost: 2100, UnitCost: 700, SaleTotal: 2100, SaleUnitPrice: 700},
			wantSubtotal:  2100,
			wantBasePrice: 2100,
			wantTotal:     2100,
		},
		{
			name: "failure rate surcharges filament+waste+energy+machine",
			in: PricingInput{
				IncludeMachineCost: true,
				MachineCostPerHour: 10,
				Items: []PricingItemInput{
					{
						Quantity:       1,
						PrintTimeHours: 1, // machine = R$10 = 1000
						CostPreset:     &CostPresetInput{FailureRatePercentage: 10},
						Filaments:      []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}, // 10000
					},
				},
			},
			// failureBase = 10000 + 0 + 0 + 1000 = 11000; failure 10% = 1100; total 12100
			wantItem:      PricingItemResult{FilamentCost: 10000, MachineCost: 1000, FailureCost: 1100, ItemTotalCost: 12100, UnitCost: 12100, SaleTotal: 12100, SaleUnitPrice: 12100},
			wantSubtotal:  12100,
			wantBasePrice: 12100,
			wantTotal:     12100,
		},
		{
			name: "waste uses preset waste_grams_per_color_change",
			in: PricingInput{
				IncludeWasteCost: true,
				Items: []PricingItemInput{
					{
						Quantity:   1,
						CostPreset: &CostPresetInput{WasteGramsPerColorChange: 20},
						Filaments: []PricingFilamentInput{
							{Grams: 100, PricePerKgCents: 12000},
							{Grams: 50, PricePerKgCents: 20000},
						},
					},
				},
			},
			// avg price = 16000; waste 20g*(2-1)=20g => round(20/1000*16000)=320; filament 2200
			wantItem:      PricingItemResult{FilamentCost: 2200, WasteCost: 320, ItemTotalCost: 2520, UnitCost: 2520, SaleTotal: 2520, SaleUnitPrice: 2520},
			wantSubtotal:  2520,
			wantBasePrice: 2520,
			wantTotal:     2520,
		},
		{
			name: "waste falls back to 15g when preset waste is zero",
			in: PricingInput{
				IncludeWasteCost: true,
				Items: []PricingItemInput{
					{
						Quantity:   1,
						CostPreset: &CostPresetInput{WasteGramsPerColorChange: 0}, // fallback 15
						Filaments: []PricingFilamentInput{
							{Grams: 100, PricePerKgCents: 12000},
							{Grams: 50, PricePerKgCents: 20000},
						},
					},
				},
			},
			// avg 16000; 15g => 240
			wantItem:      PricingItemResult{FilamentCost: 2200, WasteCost: 240, ItemTotalCost: 2440, UnitCost: 2440, SaleTotal: 2440, SaleUnitPrice: 2440},
			wantSubtotal:  2440,
			wantBasePrice: 2440,
			wantTotal:     2440,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got.Items) != 1 {
				t.Fatalf("want 1 item, got %d", len(got.Items))
			}
			if got.Items[0] != tt.wantItem {
				t.Errorf("item = %+v\nwant   %+v", got.Items[0], tt.wantItem)
			}
			if got.Subtotal != tt.wantSubtotal {
				t.Errorf("Subtotal = %d, want %d", got.Subtotal, tt.wantSubtotal)
			}
			if got.BasePrice != tt.wantBasePrice {
				t.Errorf("BasePrice = %d, want %d", got.BasePrice, tt.wantBasePrice)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}
		})
	}
}

// TestCalculate_DiscountShippingTax covers the budget-level adjustments: discount
// (percent/fixed/cap), shipping (override + computed), and the "por dentro" tax
// formula, including the exact-sum invariant with shipping.
func TestCalculate_DiscountShippingTax(t *testing.T) {
	int64p := func(v int64) *int64 { return &v }

	// A single 1000g@R$100/kg item => base 10000 cents with no preset.
	base := func() []PricingItemInput {
		return []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}}}
	}

	tests := []struct {
		name         string
		in           PricingInput
		wantBase     int64
		wantDiscount int64
		wantShipping int64
		wantTax      int64
		wantRate     float64
		wantTotal    int64
	}{
		{
			name:         "percent discount 10% of base",
			in:           PricingInput{DiscountType: DiscountTypePercent, DiscountValue: 10, Items: base()},
			wantBase:     10000,
			wantDiscount: 1000,
			wantTotal:    9000,
		},
		{
			name:         "fixed discount in reais",
			in:           PricingInput{DiscountType: DiscountTypeFixed, DiscountValue: 50, Items: base()}, // R$50 = 5000
			wantBase:     10000,
			wantDiscount: 5000,
			wantTotal:    5000,
		},
		{
			name:         "discount capped at base price",
			in:           PricingInput{DiscountType: DiscountTypeFixed, DiscountValue: 200, Items: base()}, // R$200 = 20000 > base
			wantBase:     10000,
			wantDiscount: 10000,
			wantTotal:    0,
		},
		{
			name:         "shipping override added as-is",
			in:           PricingInput{IncludeShipping: true, ShippingOverrideCents: int64p(500), Items: base()},
			wantBase:     10000,
			wantShipping: 500,
			wantTotal:    10500,
		},
		{
			name:         "shipping computed from base + per gram",
			in:           PricingInput{IncludeShipping: true, ShippingCostBase: 10, ShippingCostPerGram: 0.05, Items: base()}, // (10 + 0.05*1000)=60 => 6000
			wantBase:     10000,
			wantShipping: 6000,
			wantTotal:    16000,
		},
		{
			name:         "shipping ignored when flag off",
			in:           PricingInput{ShippingCostBase: 10, ShippingCostPerGram: 0.05, Items: base()},
			wantBase:     10000,
			wantShipping: 0,
			wantTotal:    10000,
		},
		{
			name:      "tax inside 6% grosses up net",
			in:        PricingInput{TaxRatePercent: 6, Items: []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}}}}}, // base 1000
			wantBase:  1000,
			wantTax:   64, // round(1000/0.94)=1064 => tax 64
			wantRate:  6,
			wantTotal: 1064,
		},
		{
			name: "discount + shipping + tax combined",
			in: PricingInput{
				DiscountType:          DiscountTypePercent,
				DiscountValue:         10,
				IncludeShipping:       true,
				ShippingOverrideCents: int64p(500),
				TaxRatePercent:        6,
				Items:                 base(),
			},
			// base 10000; discount 1000; net before tax = 10000-1000+500 = 9500;
			// total = round(9500/0.94) = 10106; tax = 606
			wantBase:     10000,
			wantDiscount: 1000,
			wantShipping: 500,
			wantTax:      606,
			wantRate:     6,
			wantTotal:    10106,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.BasePrice != tt.wantBase {
				t.Errorf("BasePrice = %d, want %d", got.BasePrice, tt.wantBase)
			}
			if got.DiscountAmount != tt.wantDiscount {
				t.Errorf("DiscountAmount = %d, want %d", got.DiscountAmount, tt.wantDiscount)
			}
			if got.ShippingCost != tt.wantShipping {
				t.Errorf("ShippingCost = %d, want %d", got.ShippingCost, tt.wantShipping)
			}
			if got.TaxAmount != tt.wantTax {
				t.Errorf("TaxAmount = %d, want %d", got.TaxAmount, tt.wantTax)
			}
			if got.TaxRateApplied != tt.wantRate {
				t.Errorf("TaxRateApplied = %v, want %v", got.TaxRateApplied, tt.wantRate)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}

			// Exact-sum invariant: item sale totals + shipping == total.
			var saleSum int64
			for _, it := range got.Items {
				saleSum += it.SaleTotal
			}
			if saleSum+got.ShippingCost != got.Total {
				t.Errorf("sum(SaleTotal)=%d + shipping=%d = %d, want Total %d", saleSum, got.ShippingCost, saleSum+got.ShippingCost, got.Total)
			}
		})
	}
}

// TestTaxInsideTotalCents pins the tax-inside helper, including the documented
// net 1000 @ 6% => 1064 example and the no-op edge cases.
func TestTaxInsideTotalCents(t *testing.T) {
	tests := []struct {
		net  int64
		rate float64
		want int64
	}{
		{net: 1000, rate: 6, want: 1064},
		{net: 10000, rate: 10, want: 11111}, // round(10000/0.9)=11111
		{net: 1000, rate: 0, want: 1000},
		{net: 0, rate: 6, want: 0},
		{net: 1000, rate: 100, want: 1000}, // invalid rate => unchanged
	}
	for _, tt := range tests {
		if got := TaxInsideTotalCents(tt.net, tt.rate); got != tt.want {
			t.Errorf("TaxInsideTotalCents(%d, %v) = %d, want %d", tt.net, tt.rate, got, tt.want)
		}
	}
}

// TestExactSumInvariant_WithShipping asserts, over a multi-item budget carrying
// overhead/profit, discount, shipping and tax, that the per-item sale totals plus
// shipping reconcile EXACTLY to the total.
func TestExactSumInvariant_WithShipping(t *testing.T) {
	int64p := func(v int64) *int64 { return &v }
	in := PricingInput{
		BudgetCostPreset:      &CostPresetInput{OverheadPercentage: 7, ProfitMarginPercentage: 23},
		DiscountType:          DiscountTypePercent,
		DiscountValue:         12.5,
		IncludeShipping:       true,
		ShippingOverrideCents: int64p(1337),
		TaxRatePercent:        8.25,
		Items: []PricingItemInput{
			{Quantity: 2, Filaments: []PricingFilamentInput{{Grams: 123, PricePerKgCents: 13700}}},
			{Quantity: 3, Filaments: []PricingFilamentInput{{Grams: 456, PricePerKgCents: 21900}}},
			{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 77, PricePerKgCents: 9100}}},
		},
	}
	got, err := Calculate(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var saleSum int64
	for _, it := range got.Items {
		saleSum += it.SaleTotal
	}
	if saleSum+got.ShippingCost != got.Total {
		t.Fatalf("sum(SaleTotal)=%d + shipping=%d = %d, want Total %d", saleSum, got.ShippingCost, saleSum+got.ShippingCost, got.Total)
	}
	// And base - discount + shipping + tax == total.
	if got.BasePrice-got.DiscountAmount+got.ShippingCost+got.TaxAmount != got.Total {
		t.Fatalf("base/discount/shipping/tax do not reconcile to total")
	}
}

// oldCalculate replicates the pre-Phase-4A pricing engine EXACTLY (filament, waste
// at the fixed 15g constant, energy, setup + manual labor; overhead/profit; markup
// distributed to the total). It is the reference for the backward-compatibility
// tests: with every new flag off/zero the current engine must match it byte for
// byte. The returned struct uses the current shape with new fields zeroed and
// BasePrice == Total (there are no discount/shipping/tax adjustments).
func oldCalculate(in PricingInput) PricingResult {
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
	result.BasePrice = result.Total

	saleTotals := DistributeMarkup(itemCosts, result.Total)
	for i := range result.Items {
		result.Items[i].SaleTotal = saleTotals[i]
		result.Items[i].SaleUnitPrice = UnitPriceCents(saleTotals[i], in.Items[i].Quantity)
	}
	return result
}

// TestBackwardCompat_Table proves that with every new flag off/zero the current
// engine is byte-identical to the legacy engine on a handful of representative
// inputs.
func TestBackwardCompat_Table(t *testing.T) {
	preset := &CostPresetInput{LaborRatePerHour: 60, OverheadPercentage: 10, ProfitMarginPercentage: 20}
	inputs := []PricingInput{
		{Items: []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}}}},
		{IncludeWasteCost: true, Items: []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}, {Grams: 50, PricePerKgCents: 20000}}}}},
		{EnergyEnabled: true, MachinePowerWatts: 200, EnergyPricePerKwh: 1, Items: []PricingItemInput{{Quantity: 2, PrintTimeHours: 10, CostPreset: preset, Filaments: []PricingFilamentInput{{Grams: 300, PricePerKgCents: 15000}}}}},
		{BudgetCostPreset: preset, Items: []PricingItemInput{
			{Quantity: 1, SetupTimeMinutes: 30, ManualLaborMinutesTotal: 90, CostPreset: preset, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}},
			{Quantity: 4, Filaments: []PricingFilamentInput{{Grams: 250, PricePerKgCents: 8000}}},
		}},
	}
	for i, in := range inputs {
		got, err := Calculate(in)
		if err != nil {
			t.Fatalf("case %d: unexpected error: %v", i, err)
		}
		want := oldCalculate(in)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("case %d: new engine diverged from legacy\n got=%+v\nwant=%+v", i, got, want)
		}
	}
}

// TestBackwardCompat_Differential is a randomized differential test: for many random
// budgets built with ONLY legacy inputs (new flags off, failure 0, waste configured
// to the legacy 15g either explicitly or via the zero-fallback, no machine/post-proc/
// packaging/QC/discount/shipping/tax) the current engine must equal the legacy engine.
func TestBackwardCompat_Differential(t *testing.T) {
	rng := rand.New(rand.NewSource(20260107))
	for iter := 0; iter < 2000; iter++ {
		nItems := 1 + rng.Intn(4)
		items := make([]PricingItemInput, nItems)
		for i := range items {
			nFil := 1 + rng.Intn(3)
			fils := make([]PricingFilamentInput, nFil)
			for j := range fils {
				fils[j] = PricingFilamentInput{
					Grams:           math.Round(rng.Float64()*500*100) / 100,
					PricePerKgCents: float64(5000 + rng.Intn(25000)),
				}
			}
			// A legacy-shaped cost preset: labor/overhead/profit only; waste either the
			// explicit 15g or 0 (which the engine treats as 15g). All Phase-4A rates 0.
			var cp *CostPresetInput
			if rng.Intn(2) == 0 {
				waste := 0.0
				if rng.Intn(2) == 0 {
					waste = 15
				}
				cp = &CostPresetInput{
					LaborRatePerHour:         float64(rng.Intn(120)),
					OverheadPercentage:       float64(rng.Intn(30)),
					ProfitMarginPercentage:   float64(rng.Intn(80)),
					WasteGramsPerColorChange: waste,
				}
			}
			items[i] = PricingItemInput{
				Quantity:                1 + rng.Intn(5),
				PrintTimeHours:          rng.Intn(12),
				PrintTimeMinutes:        rng.Intn(60),
				SetupTimeMinutes:        rng.Intn(120),
				ManualLaborMinutesTotal: rng.Intn(240),
				CostPreset:              cp,
				Filaments:               fils,
			}
		}

		in := PricingInput{
			IncludeWasteCost:  rng.Intn(2) == 0,
			EnergyEnabled:     rng.Intn(2) == 0,
			MachinePowerWatts: float64(100 + rng.Intn(400)),
			EnergyPricePerKwh: 0.5 + rng.Float64(),
			Items:             items,
		}
		if rng.Intn(2) == 0 {
			in.BudgetCostPreset = &CostPresetInput{OverheadPercentage: float64(rng.Intn(25)), ProfitMarginPercentage: float64(rng.Intn(100))}
		}

		got, err := Calculate(in)
		if err != nil {
			t.Fatalf("iter %d: unexpected error: %v", iter, err)
		}
		want := oldCalculate(in)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iter %d: new engine diverged from legacy\n got=%+v\nwant=%+v\nin=%+v", iter, got, want, in)
		}
	}
}
