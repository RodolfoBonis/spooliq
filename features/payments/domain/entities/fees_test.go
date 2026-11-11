package entities

import "testing"

func ratesFixture() *FeeRates {
	return &FeeRates{
		Pix:      0,
		Credit1x: 4.2,
		CreditInstallments: map[string]float64{
			"2": 5.4,
			"3": 6.1,
			"4": 7.9,
		},
	}
}

func TestFeeFor(t *testing.T) {
	rates := ratesFixture()
	tests := []struct {
		name         string
		rates        *FeeRates
		method       string
		installments int
		paid         int64
		wantRate     float64
		wantCents    int64
		wantOK       bool
	}{
		{
			// The real first payment: R$ 217,12 charged in 4x at 7,9% -> R$ 17,15 fee,
			// net R$ 199,97.
			name:  "real case: 21712 at 7.9% -> 1715, net 19997",
			rates: rates, method: "credit_card", installments: 4, paid: 21712,
			wantRate: 7.9, wantCents: 1715, wantOK: true,
		},
		{
			name:  "pix uses the pix rate (0%)",
			rates: rates, method: "pix", installments: 1, paid: 10000,
			wantRate: 0, wantCents: 0, wantOK: true,
		},
		{
			name:  "credit 1x uses the a vista rate",
			rates: rates, method: "credit_card", installments: 1, paid: 10000,
			wantRate: 4.2, wantCents: 420, wantOK: true,
		},
		{
			name:  "credit 0 installments falls back to a vista",
			rates: rates, method: "credit_card", installments: 0, paid: 10000,
			wantRate: 4.2, wantCents: 420, wantOK: true,
		},
		{
			name:  "credit 2x uses the installment rate",
			rates: rates, method: "credit_card", installments: 2, paid: 10000,
			wantRate: 5.4, wantCents: 540, wantOK: true,
		},
		{
			name:  "unknown installment -> not ok",
			rates: rates, method: "credit_card", installments: 12, paid: 10000,
			wantOK: false,
		},
		{
			name:  "nil table -> not ok",
			rates: nil, method: "pix", installments: 1, paid: 10000,
			wantOK: false,
		},
		{
			name:  "rounds half away from zero",
			rates: &FeeRates{Credit1x: 5}, method: "credit_card", installments: 1, paid: 101,
			wantRate: 5, wantCents: 5, wantOK: true, // 5.05 -> 5
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, cents, ok := FeeFor(tt.rates, tt.method, tt.installments, tt.paid)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if rate != tt.wantRate {
				t.Errorf("rate = %v, want %v", rate, tt.wantRate)
			}
			if cents != tt.wantCents {
				t.Errorf("cents = %d, want %d", cents, tt.wantCents)
			}
			if tt.paid-cents != tt.paid-tt.wantCents {
				t.Errorf("net mismatch")
			}
		})
	}
}

func TestValidateFeeRates(t *testing.T) {
	tests := []struct {
		name    string
		rates   *FeeRates
		wantErr bool
	}{
		{name: "nil is valid", rates: nil},
		{name: "valid table", rates: ratesFixture()},
		{name: "rate above 30", rates: &FeeRates{Credit1x: 30.01}, wantErr: true},
		{name: "negative rate", rates: &FeeRates{Pix: -1}, wantErr: true},
		{name: "three decimals", rates: &FeeRates{Credit1x: 4.255}, wantErr: true},
		{name: "bad installment key", rates: &FeeRates{CreditInstallments: map[string]float64{"13": 5}}, wantErr: true},
		{name: "installment key 1 is invalid (use credit_1x)", rates: &FeeRates{CreditInstallments: map[string]float64{"1": 5}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeeRates(tt.rates)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReferenceRate(t *testing.T) {
	rates := ratesFixture()
	if r, ok := rates.ReferenceRate(3); !ok || r != 6.1 {
		t.Errorf("ReferenceRate(3) = %v,%v want 6.1,true", r, ok)
	}
	if r, ok := rates.ReferenceRate(1); !ok || r != 4.2 {
		t.Errorf("ReferenceRate(1) = %v,%v want 4.2,true", r, ok)
	}
	if _, ok := rates.ReferenceRate(12); ok {
		t.Errorf("ReferenceRate(12) should be unknown")
	}
	var nilRates *FeeRates
	if _, ok := nilRates.ReferenceRate(1); ok {
		t.Errorf("nil ReferenceRate should be unknown")
	}
}

func TestFeeFor_MethodRecognition(t *testing.T) {
	rates := ratesFixture()
	tests := []struct {
		name         string
		method       string
		installments int
		wantOK       bool
		wantRate     float64
	}{
		{name: "pix lowercase", method: "pix", installments: 1, wantOK: true, wantRate: 0},
		{name: "PIX uppercase (EqualFold)", method: "PIX", installments: 1, wantOK: true, wantRate: 0},
		{name: "credit_card lowercase", method: "credit_card", installments: 1, wantOK: true, wantRate: 4.2},
		{name: "Credit_Card mixed case (EqualFold)", method: "Credit_Card", installments: 1, wantOK: true, wantRate: 4.2},
		{name: "empty method is unknown", method: "", installments: 1, wantOK: false},
		{name: "unknown method (boleto) is unknown", method: "boleto", installments: 1, wantOK: false},
		{name: "debit is unknown (not a known method)", method: "debit_card", installments: 1, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, _, ok := FeeFor(rates, tt.method, tt.installments, 10000)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && rate != tt.wantRate {
				t.Errorf("rate = %v, want %v", rate, tt.wantRate)
			}
		})
	}
}
