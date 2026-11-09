package entities

import "testing"

func pct(v float64) *float64 { return &v }

func TestDepositAmountHalfUp(t *testing.T) {
	cases := []struct {
		name    string
		total   int64
		percent float64
		want    int64
	}{
		{"50% of 200", 200, 50, 100},
		{"50% of 201 rounds half-up", 201, 50, 101},
		{"30% of 1000", 1000, 30, 300},
		{"33% of 100 -> 33", 100, 33, 33},
		{"zero percent", 1000, 0, 0},
		{"negative percent", 1000, -5, 0},
		{"zero total", 0, 50, 0},
		{"fractional percent 12.5% of 100 -> 13", 100, 12.5, 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DepositAmount(tc.total, tc.percent); got != tc.want {
				t.Fatalf("DepositAmount(%d,%v) = %d want %d", tc.total, tc.percent, got, tc.want)
			}
		})
	}
}

func TestNextChargeFor(t *testing.T) {
	cases := []struct {
		name       string
		total      int64
		paid       int64
		deposit    *float64
		wantKind   ChargeKind
		wantAmount int64
		wantOK     bool
		wantReason string
	}{
		{"nothing paid, deposit 50%", 2000, 0, pct(50), KindDeposit, 1000, true, ""},
		{"nothing paid, no deposit", 2000, 0, nil, KindFull, 2000, true, ""},
		{"partial paid -> balance", 2000, 1000, pct(50), KindBalance, 1000, true, ""},
		{"fully paid", 2000, 2000, pct(50), KindFull, 0, false, ReasonAlreadyPaid},
		{"overpaid counts as paid", 2000, 2500, pct(50), KindFull, 0, false, ReasonAlreadyPaid},
		{"deposit below R$1 -> full", 150, 0, pct(10), KindFull, 150, true, ""},
		{"remainder below R$1 -> full", 150, 0, pct(95), KindFull, 150, true, ""},
		{"balance below R$1 -> not chargeable", 2050, 2000, nil, KindBalance, 50, false, ReasonBelowMinimum},
		{"total below R$1 no deposit", 50, 0, nil, KindFull, 50, false, ReasonBelowMinimum},
		{"total below R$1 WITH deposit -> still not chargeable", 50, 0, pct(50), KindFull, 50, false, ReasonBelowMinimum},
		{"zero total", 0, 0, pct(50), KindFull, 0, false, ReasonInvalidTotal},
		{"deposit exactly R$1, remainder ok", 200, 0, pct(50), KindDeposit, 100, true, ""},
		{"deposit 100% -> full", 2000, 0, pct(100), KindFull, 2000, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NextChargeFor(tc.total, tc.paid, tc.deposit)
			if got.Kind != tc.wantKind || got.Amount != tc.wantAmount || got.Chargeable != tc.wantOK || got.Reason != tc.wantReason {
				t.Fatalf("NextChargeFor(%d,%d,%v) = %+v want kind=%s amount=%d ok=%v reason=%q",
					tc.total, tc.paid, tc.deposit, got, tc.wantKind, tc.wantAmount, tc.wantOK, tc.wantReason)
			}
		})
	}
}

func TestPixPrice(t *testing.T) {
	cases := []struct {
		amount  int64
		percent int
		want    int64
	}{
		{1000, 5, 950},
		{1000, 0, 1000},
		{201, 50, 100}, // discount 100.5 -> 101; 201-101 = 100
		{1000, -5, 1000},
		{0, 10, 0},
		{2000, 10, 1800},
	}
	for _, tc := range cases {
		if got := PixPrice(tc.amount, tc.percent); got != tc.want {
			t.Fatalf("PixPrice(%d,%d) = %d want %d", tc.amount, tc.percent, got, tc.want)
		}
	}
}

func TestInterestFreeInstallment(t *testing.T) {
	cases := []struct {
		amount int64
		n      int
		want   int64
	}{
		{1200, 1, 1200},
		{1200, 3, 400},
		{1000, 3, 333},
		{1000, 12, 83},
		{100, 12, 8},
		{1200, 0, 1200},
		{0, 3, 0},
	}
	for _, tc := range cases {
		if got := InterestFreeInstallment(tc.amount, tc.n); got != tc.want {
			t.Fatalf("InterestFreeInstallment(%d,%d) = %d want %d", tc.amount, tc.n, got, tc.want)
		}
	}
}

func TestStatus(t *testing.T) {
	cases := []struct {
		total, paid int64
		want        PaymentStatus
	}{
		{2000, 0, StatusUnpaid},
		{2000, 1000, StatusPartial},
		{2000, 2000, StatusPaid},
		{2000, 2500, StatusPaid},
		{2000, -1, StatusUnpaid},
	}
	for _, tc := range cases {
		if got := Status(tc.total, tc.paid); got != tc.want {
			t.Fatalf("Status(%d,%d) = %s want %s", tc.total, tc.paid, got, tc.want)
		}
	}
}
