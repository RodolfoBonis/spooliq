package services

import "math"

// SupplyCostCents computes the cost in cents of a single supply line attached to a
// budget item: quantityPerPiece units consumed per printed piece, times the number
// of printed pieces (productQuantity), times the supply's unit cost (in cents),
// rounded to the nearest cent.
//
// It is the single source of truth shared by the pricing engine (item SuppliesCost
// = sum of its lines) and the API responses (each line's `total`), so a supply
// line's displayed total always reconciles with the item's supplies cost.
//
// The product is computed in float64 and clamped to [0, math.MaxInt64] before the
// int64 conversion: a float64 outside the int64 range converts to an
// implementation-defined value in Go, so even though the request validation bounds
// quantity_per_piece and product_quantity, an extreme unit_cost is guarded here
// rather than silently wrapping to a negative/garbage cost.
func SupplyCostCents(quantityPerPiece float64, productQuantity int, unitCostCents int64) int64 {
	if quantityPerPiece <= 0 || productQuantity <= 0 || unitCostCents <= 0 {
		return 0
	}
	cents := math.Round(quantityPerPiece * float64(productQuantity) * float64(unitCostCents))
	if cents >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if cents <= 0 {
		return 0
	}
	return int64(cents)
}
