package services_test

import (
	"testing"

	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
)

// TestSupplyCostCents verifies the per-line supply cost rounding: qtyPerPiece *
// productQuantity * unitCostCents, rounded to the nearest cent, with zero guards.
func TestSupplyCostCents(t *testing.T) {
	cases := []struct {
		name         string
		qtyPerPiece  float64
		productQty   int
		unitCostCent int64
		want         int64
	}{
		{"one LED per piece, 3 pieces @ R$1,50", 1, 3, 150, 450},
		{"half a unit per piece rounds", 0.5, 3, 101, 152}, // 0.5*3*101 = 151.5 -> 152
		{"fractional qty", 0.33, 10, 100, 330},             // 0.33*10*100 = 330
		{"zero quantity per piece", 0, 5, 100, 0},          // guard
		{"zero product quantity", 2, 0, 100, 0},            // guard
		{"zero unit cost", 2, 5, 0, 0},                     // guard
		{"negative guard", -1, 5, 100, 0},                  // guard
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pricing.SupplyCostCents(tc.qtyPerPiece, tc.productQty, tc.unitCostCent)
			if got != tc.want {
				t.Fatalf("SupplyCostCents(%v,%d,%d) = %d, want %d", tc.qtyPerPiece, tc.productQty, tc.unitCostCent, got, tc.want)
			}
		})
	}
}

// TestCalculate_SuppliesInItemCostAndMarkup proves supplies are a direct item cost
// multiplied by the product quantity, added to the item total, summed into the
// budget SuppliesCost, and receive the budget markup (overhead + profit) like any
// other direct cost.
func TestCalculate_SuppliesInItemCostAndMarkup(t *testing.T) {
	// 1 filament line (100g @ R$100/kg = R$10,00 = 1000c) + 1 supply (2 units/piece,
	// 3 pieces, R$1,00/unit = 100c) => supplies = 2*3*100 = 600c.
	cp := &pricing.CostPresetInput{OverheadPercentage: 10, ProfitMarginPercentage: 20}
	res, err := pricing.Calculate(pricing.PricingInput{
		BudgetCostPreset: cp,
		Items: []pricing.PricingItemInput{
			{
				Quantity:  3,
				Filaments: []pricing.PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}},
				Supplies:  []pricing.PricingSupplyInput{{QuantityPerPiece: 2, UnitCostCents: 100}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	if res.Items[0].SuppliesCost != 600 {
		t.Fatalf("item SuppliesCost = %d, want 600", res.Items[0].SuppliesCost)
	}
	if res.SuppliesCost != 600 {
		t.Fatalf("budget SuppliesCost = %d, want 600", res.SuppliesCost)
	}
	// filament 1000 + supplies 600 = 1600 item total.
	if res.Items[0].ItemTotalCost != 1600 {
		t.Fatalf("item total = %d, want 1600 (1000 filament + 600 supplies)", res.Items[0].ItemTotalCost)
	}
	if res.Subtotal != 1600 {
		t.Fatalf("subtotal = %d, want 1600", res.Subtotal)
	}
	// Overhead 10% of 1600 = 160; profit 20% of (1600+160)=1760 => 352; base 2112.
	if res.Overhead != 160 || res.Profit != 352 || res.BasePrice != 2112 {
		t.Fatalf("markup = overhead %d / profit %d / base %d, want 160/352/2112", res.Overhead, res.Profit, res.BasePrice)
	}
	if res.Total != 2112 {
		t.Fatalf("total = %d, want 2112 (markup applied to supplies too)", res.Total)
	}
	// The whole item (incl. supplies) carries the full markup: SaleTotal == total.
	if res.Items[0].SaleTotal != 2112 {
		t.Fatalf("item SaleTotal = %d, want 2112", res.Items[0].SaleTotal)
	}
}

// TestCalculate_SuppliesExcludedFromFailureBase proves supplies do NOT inflate the
// failure-rate surcharge (a supply is not reprinted on a failed job): the failure
// cost is computed only from filament+waste+energy+machine.
func TestCalculate_SuppliesExcludedFromFailureBase(t *testing.T) {
	// 100g @ R$100/kg = 1000c filament, 50% failure rate, plus a big supply cost.
	cp := &pricing.CostPresetInput{FailureRatePercentage: 50}

	withSupply, err := pricing.Calculate(pricing.PricingInput{
		Items: []pricing.PricingItemInput{
			{
				Quantity:   1,
				CostPreset: cp,
				Filaments:  []pricing.PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}},
				Supplies:   []pricing.PricingSupplyInput{{QuantityPerPiece: 1, UnitCostCents: 100000}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}

	// Failure is 50% of the 1000c filament only => 500c, regardless of the 100000c supply.
	if withSupply.FailureCost != 500 {
		t.Fatalf("failure cost = %d, want 500 (supplies must be excluded from the failure base)", withSupply.FailureCost)
	}
	if withSupply.Items[0].SuppliesCost != 100000 {
		t.Fatalf("item SuppliesCost = %d, want 100000", withSupply.Items[0].SuppliesCost)
	}
	// Item total = filament 1000 + failure 500 + supplies 100000 = 101500.
	if withSupply.Items[0].ItemTotalCost != 101500 {
		t.Fatalf("item total = %d, want 101500", withSupply.Items[0].ItemTotalCost)
	}
}
