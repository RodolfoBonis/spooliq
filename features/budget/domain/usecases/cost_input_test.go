package usecases

import "testing"

func TestResolveIncludeMachineCost(t *testing.T) {
	if !resolveIncludeMachineCost(nil) {
		t.Errorf("nil should default to true (new budgets charge machine cost)")
	}
	f := false
	if resolveIncludeMachineCost(&f) {
		t.Errorf("explicit false must be honored")
	}
	tr := true
	if !resolveIncludeMachineCost(&tr) {
		t.Errorf("explicit true must be honored")
	}
}

func TestValidateDiscountInput(t *testing.T) {
	str := func(s string) *string { return &s }
	fl := func(v float64) *float64 { return &v }

	tests := []struct {
		name    string
		dType   *string
		dValue  *float64
		wantErr bool
	}{
		{name: "both nil is valid (no discount)", dType: nil, dValue: nil, wantErr: false},
		{name: "percent within range", dType: str("percent"), dValue: fl(10), wantErr: false},
		{name: "percent at bounds 0", dType: str("percent"), dValue: fl(0), wantErr: false},
		{name: "percent at bounds 100", dType: str("percent"), dValue: fl(100), wantErr: false},
		{name: "percent over 100 rejected", dType: str("percent"), dValue: fl(100.01), wantErr: true},
		{name: "percent negative rejected", dType: str("percent"), dValue: fl(-1), wantErr: true},
		{name: "fixed non-negative valid", dType: str("fixed"), dValue: fl(50), wantErr: false},
		{name: "fixed negative rejected", dType: str("fixed"), dValue: fl(-1), wantErr: true},
		{name: "type without value rejected", dType: str("percent"), dValue: nil, wantErr: true},
		{name: "value without type rejected", dType: nil, dValue: fl(10), wantErr: true},
		{name: "unknown type rejected", dType: str("bogus"), dValue: fl(10), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDiscountInput(tt.dType, tt.dValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDiscountInput(%v, %v) err=%v, wantErr=%v", tt.dType, tt.dValue, err, tt.wantErr)
			}
		})
	}
}
