package services_test

import (
	"testing"

	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
)

// TestSupplyCostCents verifies the per-line PACKAGE supply cost rounding:
// round(qtyPerPiece * productQuantity * packagePriceCents / packageSize), with a
// single math.Round per line and the zero/guard cases.
func TestSupplyCostCents(t *testing.T) {
	cases := []struct {
		name         string
		qtyPerPiece  float64
		productQty   int
		packagePrice int64
		packageSize  float64
		want         int64
	}{
		// Lotion: 400 ml for R$69,90 (6990c), 10 ml/piece on 10 pieces.
		// 10*10*6990/400 = 1747.5 -> 1748.
		{"lotion 400ml package rounds half up", 10, 10, 6990, 400, 1748},
		// LED: pack of 100 LEDs for R$50,00 (5000c), 2/piece on 10 pieces.
		// 2*10*5000/100 = 1000.
		{"LED pack exact", 2, 10, 5000, 100, 1000},
		// Legacy (package_size 1, package_price = old unit_cost) matches the pre-package
		// result qty*productQty*unitCost: 1*3*150 = 450; 0.5*3*101 = 151.5 -> 152.
		{"legacy size 1 equals old result", 1, 3, 150, 1, 450},
		{"legacy size 1 half rounds", 0.5, 3, 101, 1, 152},
		// packageSize <= 0 is treated as 1 (guard, never divides by zero).
		{"zero package size treated as 1", 1, 3, 150, 0, 450},
		{"negative package size treated as 1", 1, 3, 150, -5, 450},
		{"zero quantity per piece", 0, 5, 100, 10, 0},
		{"zero product quantity", 2, 0, 100, 10, 0},
		{"zero package price", 2, 5, 0, 10, 0},
		{"negative quantity guard", -1, 5, 100, 10, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pricing.SupplyCostCents(tc.qtyPerPiece, tc.productQty, tc.packagePrice, tc.packageSize)
			if got != tc.want {
				t.Fatalf("SupplyCostCents(%v,%d,%d,%v) = %d, want %d", tc.qtyPerPiece, tc.productQty, tc.packagePrice, tc.packageSize, got, tc.want)
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
	// 3 pieces, package of 1 unit @ R$1,00 = 100c) => supplies = 2*3*100/1 = 600c.
	cp := &pricing.CostPresetInput{OverheadPercentage: 10, ProfitMarginPercentage: 20}
	res, err := pricing.Calculate(pricing.PricingInput{
		BudgetCostPreset: cp,
		Items: []pricing.PricingItemInput{
			{
				Quantity:  3,
				Filaments: []pricing.PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}},
				Supplies:  []pricing.PricingSupplyInput{{QuantityPerPiece: 2, PackagePriceCents: 100, PackageSize: 1}},
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

// TestCalculate_SuppliesPackagePricing proves the package formula flows through the
// engine: a 400 ml lotion package (R$69,90 = 6990c) used 10 ml/piece on 10 pieces
// costs 1748c (round(10*10*6990/400)), summed into the item and budget supplies cost.
func TestCalculate_SuppliesPackagePricing(t *testing.T) {
	res, err := pricing.Calculate(pricing.PricingInput{
		Items: []pricing.PricingItemInput{
			{
				Quantity: 10,
				Supplies: []pricing.PricingSupplyInput{{QuantityPerPiece: 10, PackagePriceCents: 6990, PackageSize: 400}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	if res.Items[0].SuppliesCost != 1748 {
		t.Fatalf("item SuppliesCost = %d, want 1748", res.Items[0].SuppliesCost)
	}
	if res.SuppliesCost != 1748 {
		t.Fatalf("budget SuppliesCost = %d, want 1748", res.SuppliesCost)
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
				Supplies:   []pricing.PricingSupplyInput{{QuantityPerPiece: 1, PackagePriceCents: 100000, PackageSize: 1}},
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
