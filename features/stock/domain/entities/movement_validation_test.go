package entities

import "testing"

// TestNormalize_SignAndValidation is the movement validation table: it proves the
// per-type sign handling (waste is stored negative, purchase stays positive,
// adjustment keeps its sign) and the rejection rules (bad grams, non-manual types).
func TestNormalize_SignAndValidation(t *testing.T) {
	cases := []struct {
		name       string
		req        CreateMovementRequest
		wantSigned int64
		wantCode   string
	}{
		{"purchase positive", CreateMovementRequest{Type: MovementPurchase, Grams: 1000}, 1000, ""},
		{"purchase zero rejected", CreateMovementRequest{Type: MovementPurchase, Grams: 0}, 0, CodeInvalidMovementGrams},
		{"purchase negative rejected", CreateMovementRequest{Type: MovementPurchase, Grams: -5}, 0, CodeInvalidMovementGrams},
		{"adjustment positive", CreateMovementRequest{Type: MovementAdjustment, Grams: 250}, 250, ""},
		{"adjustment negative kept", CreateMovementRequest{Type: MovementAdjustment, Grams: -250}, -250, ""},
		{"adjustment zero rejected", CreateMovementRequest{Type: MovementAdjustment, Grams: 0}, 0, CodeInvalidMovementGrams},
		{"waste stored negative", CreateMovementRequest{Type: MovementWaste, Grams: 300}, -300, ""},
		{"waste zero rejected", CreateMovementRequest{Type: MovementWaste, Grams: 0}, 0, CodeInvalidMovementGrams},
		{"waste negative input rejected", CreateMovementRequest{Type: MovementWaste, Grams: -300}, 0, CodeInvalidMovementGrams},
		{"consumption not manual", CreateMovementRequest{Type: MovementConsumption, Grams: 100}, 0, CodeInvalidMovementType},
		{"unknown type", CreateMovementRequest{Type: MovementType("bogus"), Grams: 100}, 0, CodeInvalidMovementType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			signed, code := tc.req.Normalize()
			if code != tc.wantCode {
				t.Fatalf("code = %q, want %q", code, tc.wantCode)
			}
			if code == "" && signed != tc.wantSigned {
				t.Fatalf("signed = %d, want %d", signed, tc.wantSigned)
			}
		})
	}
}

// TestIsManual documents that only purchase/adjustment/waste are user-creatable.
func TestIsManual(t *testing.T) {
	for _, mt := range []MovementType{MovementPurchase, MovementAdjustment, MovementWaste} {
		if !mt.IsManual() {
			t.Errorf("%s should be manual", mt)
		}
	}
	for _, mt := range []MovementType{MovementConsumption, MovementType("x")} {
		if mt.IsManual() {
			t.Errorf("%s should not be manual", mt)
		}
	}
}

func TestMovementType_IsKnown(t *testing.T) {
	for _, tc := range []struct {
		in   MovementType
		want bool
	}{{MovementPurchase, true}, {MovementAdjustment, true}, {MovementWaste, true}, {MovementConsumption, true}, {"bogus", false}, {"", false}} {
		if got := tc.in.IsKnown(); got != tc.want {
			t.Errorf("IsKnown(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
