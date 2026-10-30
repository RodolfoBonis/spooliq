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
func SupplyCostCents(quantityPerPiece float64, productQuantity int, unitCostCents int64) int64 {
	if quantityPerPiece <= 0 || productQuantity <= 0 || unitCostCents <= 0 {
		return 0
	}
	return int64(math.Round(quantityPerPiece * float64(productQuantity) * float64(unitCostCents)))
}
