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
	// GOLDEN values captured from main (the fee-less engine) for the representative
	// inputs. They are hard-coded (not recomputed from the engine under test) so this
	// test fails if the fee change shifts any total, tax or per-item sale value when no
	// fee is embedded — the money-safety guarantee for existing budgets.
	golden := []struct {
		total int64
		tax   int64
		sales []int64
	}{
		{total: 1200, tax: 0, sales: []int64{1200}},
		{total: 2287, tax: 137, sales: []int64{2287}},
		{total: 8569, tax: 728, sales: []int64{8569}},
		{total: 6698, tax: 804, sales: []int64{1921, 2277}},
	}
	inputs := representativePricingInputs()
	if len(inputs) != len(golden) {
		t.Fatalf("golden table out of sync: %d inputs vs %d golden", len(inputs), len(golden))
	}
	for i, in := range inputs {
		in.PaymentFeeRatePercent = 0 // explicit: no embedded fee
		got, err := Calculate(in)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if got.Total != golden[i].total {
			t.Errorf("case %d: Total = %d, want %d", i, got.Total, golden[i].total)
		}
		if got.TaxAmount != golden[i].tax {
			t.Errorf("case %d: TaxAmount = %d, want %d", i, got.TaxAmount, golden[i].tax)
		}
		if got.PaymentFee != 0 || got.PaymentFeeRateApplied != 0 {
			t.Errorf("case %d: fee must be zero, got %d / %v", i, got.PaymentFee, got.PaymentFeeRateApplied)
		}
		if len(got.Items) != len(golden[i].sales) {
			t.Fatalf("case %d: %d items, want %d", i, len(got.Items), len(golden[i].sales))
		}
		for j := range golden[i].sales {
			if got.Items[j].SaleTotal != golden[i].sales[j] {
				t.Errorf("case %d item %d: SaleTotal = %d, want %d", i, j, got.Items[j].SaleTotal, golden[i].sales[j])
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
		// With no tax, the whole gross-up is the fee: tax must be exactly 0.
		if got.TaxAmount != 0 {
			t.Errorf("tax = %d, want 0 (no tax configured)", got.TaxAmount)
		}
		// Reconciliation: net + tax + fee == total.
		if 10000+got.TaxAmount+got.PaymentFee != got.Total {
			t.Errorf("reconcile failed: %d + %d + %d != %d", 10000, got.TaxAmount, got.PaymentFee, got.Total)
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
