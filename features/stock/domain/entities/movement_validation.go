package entities

import "math"

// Validation error codes (snake_case, stable; messages are pt-BR in the use case).
const (
	// CodeInvalidMovementType is returned when the movement type is not one of the
	// user-creatable types (purchase/adjustment/waste).
	CodeInvalidMovementType = "invalid_movement_type"
	// CodeInvalidMovementGrams is returned when grams violate the rule for the type.
	CodeInvalidMovementGrams = "invalid_movement_grams"
)

// Normalize validates the request and returns the SIGNED grams delta to apply to
// the filament balance, together with an error code ("" when valid).
//
// Rules by type:
//   - purchase:   grams > 0            -> stored +grams
//   - adjustment: grams != 0 (signed)  -> stored as-is
//   - waste:      input grams > 0      -> stored -grams
//   - anything else (incl. consumption, which is system-only) -> invalid_movement_type
func (r CreateMovementRequest) Normalize() (signedGrams int64, errCode string) {
	if !r.Type.IsManual() {
		return 0, CodeInvalidMovementType
	}
	grams := int64(math.Round(r.Grams))

	switch r.Type {
	case MovementPurchase:
		if grams <= 0 {
			return 0, CodeInvalidMovementGrams
		}
		return grams, ""
	case MovementAdjustment:
		if grams == 0 {
			return 0, CodeInvalidMovementGrams
		}
		return grams, ""
	case MovementWaste:
		if grams <= 0 {
			return 0, CodeInvalidMovementGrams
		}
		return -grams, ""
	default:
		return 0, CodeInvalidMovementType
	}
}
