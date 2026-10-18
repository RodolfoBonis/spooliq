package usecases

import (
	"testing"

	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
)

func TestApplyThresholdUpdate(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	cases := []struct {
		raw     string
		start   *int
		want    *int
		wantErr bool
	}{
		{"", intPtr(50), intPtr(50), false},     // absent: unchanged
		{"null", intPtr(50), nil, false},        // cleared
		{"-1", intPtr(50), nil, false},          // cleared
		{"100", nil, intPtr(100), false},        // integer
		{"100.0", nil, intPtr(100), false},      // desktop client sends doubles
		{"99.6", nil, intPtr(100), false},       // rounded to whole grams
		{`"abc"`, intPtr(50), intPtr(50), true}, // invalid
	}
	for _, tc := range cases {
		f := &filamentEntities.FilamentEntity{LowStockThresholdGrams: tc.start}
		err := applyThresholdUpdate(f, []byte(tc.raw))
		if (err != nil) != tc.wantErr {
			t.Fatalf("%q: err = %v", tc.raw, err)
		}
		got := f.LowStockThresholdGrams
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%q: threshold = %v, want %v", tc.raw, got, tc.want)
		}
	}
}
