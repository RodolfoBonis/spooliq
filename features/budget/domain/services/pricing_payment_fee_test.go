package services

import "testing"

// representativePricingInputs returns a set of budgets that together exercise tax,
// discount, shipping and multiple items — the regression surface the payment-fee
// change must not alter when the fee rate is 0.
func representativePricingInputs() []PricingInput {
	preset := &CostPresetInput{LaborRatePerHour: 60, OverheadPercentage: 10, ProfitMarginPercentage: 20}
	shipPreset := &CostPresetInput{OverheadPercentage: 10, ProfitMarginPercentage: 20}
	return []PricingInput{
		{ // plain single item
			Items: []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 100, PricePerKgCents: 12000}}}},
		},
		{ // tax only
			TaxRatePercent: 6,
			Items:          []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 215, PricePerKgCents: 10000}}}},
		},
		{ // tax + percent discount + labor + overhead/profit
			TaxRatePercent: 8.5, DiscountType: DiscountTypePercent, DiscountValue: 10,
			BudgetCostPreset: preset,
			Items:            []PricingItemInput{{Quantity: 2, SetupTimeMinutes: 30, Filaments: []PricingFilamentInput{{Grams: 300, PricePerKgCents: 12000}}, CostPreset: preset}},
		},
		{ // tax + fixed discount + shipping override + multiple items
			TaxRatePercent: 12, DiscountType: DiscountTypeFixed, DiscountValue: 5,
			IncludeShipping: true, ShippingOverrideCents: ptrInt64(2500),
			BudgetCostPreset: shipPreset,
			Items: []PricingItemInput{
				{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 150, PricePerKgCents: 9000}}, CostPreset: shipPreset},
				{Quantity: 3, Filaments: []PricingFilamentInput{{Grams: 80, PricePerKgCents: 20000}}, CostPreset: shipPreset},
			},
		},
	}
}

func ptrInt64(v int64) *int64 { return &v }

// TestCalculate_FeeZeroIsIdentical asserts the whole result is byte-identical whether
// the payment-fee fields are absent (0) across representative budgets — the money-safety
// guarantee that existing budgets never reprice.
func TestCalculate_FeeZeroIsIdentical(t *testing.T) {
	for i, in := range representativePricingInputs() {
		withZero := in
		withZero.PaymentFeeRatePercent = 0

		base, err := Calculate(in)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		got, err := Calculate(withZero)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got.Total != base.Total || got.TaxAmount != base.TaxAmount ||
			got.PaymentFee != 0 || got.PaymentFeeRateApplied != 0 {
			t.Errorf("case %d: fee=0 changed result: %+v vs %+v", i, got, base)
		}
		// Per-item sale totals must be unchanged too.
		for j := range base.Items {
			if got.Items[j].SaleTotal != base.Items[j].SaleTotal {
				t.Errorf("case %d item %d: SaleTotal %d != %d", i, j, got.Items[j].SaleTotal, base.Items[j].SaleTotal)
			}
		}
	}
}

// TestCalculate_FeeEmbedded checks the combined tax+fee gross-up: total, the split
// between tax and fee, and rounding.
func TestCalculate_FeeEmbedded(t *testing.T) {
	// net = 10000 (single filament 1kg at R$100/kg, no markup).
	base := PricingInput{
		Items: []PricingItemInput{{Quantity: 1, Filaments: []PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}}},
	}

	t.Run("fee only", func(t *testing.T) {
		in := base
		in.PaymentFeeRatePercent = 6.1
		got, err := Calculate(in)
		if err != nil {
			t.Fatal(err)
		}
		// total = round(10000 / (1 - 6.1/100)) = round(10000/0.939) = 10650
		if got.Total != 10650 {
			t.Errorf("total = %d, want 10650", got.Total)
		}
		// fee = round(10650 * 6.1/100) = round(649.65) = 650
		if got.PaymentFee != 650 {
			t.Errorf("fee = %d, want 650", got.PaymentFee)
		}
		if got.PaymentFeeRateApplied != 6.1 {
			t.Errorf("rate applied = %v, want 6.1", got.PaymentFeeRateApplied)
		}
		// tax = total - net - fee = 10649 - 10000 - 650 = -1 ... guard: tax should be 0 here
		if got.TaxAmount != got.Total-10000-got.PaymentFee {
			t.Errorf("tax split inconsistent: %d", got.TaxAmount)
		}
	})

	t.Run("tax and fee combined", func(t *testing.T) {
		in := base
		in.TaxRatePercent = 6
		in.PaymentFeeRatePercent = 6.1
		got, err := Calculate(in)
		if err != nil {
			t.Fatal(err)
		}
		// combined = 12.1%; total = round(10000 / 0.879) = 11377
		if got.Total != 11377 {
			t.Errorf("total = %d, want 11377", got.Total)
		}
		// fee = round(11377 * 6.1/100) = round(693.997) = 694
		if got.PaymentFee != 694 {
			t.Errorf("fee = %d, want 694", got.PaymentFee)
		}
		// tax = total - net - fee = 11377 - 10000 - 694 = 683
		if got.TaxAmount != 683 {
			t.Errorf("tax = %d, want 683", got.TaxAmount)
		}
		// reconciliation: net + tax + fee == total
		if 10000+got.TaxAmount+got.PaymentFee != got.Total {
			t.Errorf("reconcile failed")
		}
	})

	t.Run("protection: tax+fee >= 100 drops the fee", func(t *testing.T) {
		in := base
		in.TaxRatePercent = 95
		in.PaymentFeeRatePercent = 6
		got, err := Calculate(in)
		if err != nil {
			t.Fatal(err)
		}
		if got.PaymentFeeRateApplied != 0 || got.PaymentFee != 0 {
			t.Errorf("fee should be dropped, got rate %v fee %d", got.PaymentFeeRateApplied, got.PaymentFee)
		}
		// tax alone still applies (95%): total = round(10000/0.05) = 200000
		if got.Total != 200000 {
			t.Errorf("total = %d, want 200000", got.Total)
		}
	})
}
