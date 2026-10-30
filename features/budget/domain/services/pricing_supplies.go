package services

import "math"

// SupplyCostCents computes the cost in cents of a single supply line attached to a
// budget item under PACKAGE pricing: a supply is bought as a package (packageSize
// units of its measurement unit for packagePriceCents cents), and each printed
// piece consumes quantityPerPiece units. The line cost for the whole item is
//
//	round(quantityPerPiece × productQuantity × packagePriceCents ÷ packageSize)
//
// so e.g. a 400 ml lotion bought for R$69,90 (6990c), used 10 ml per piece on 10
// pieces, costs round(10 × 10 × 6990 ÷ 400) = round(1747.5) = 1748c. A legacy
// supply with packageSize 1 and packagePrice equal to its old unit cost yields
// exactly the pre-package result (quantityPerPiece × productQuantity × unitCost).
//
// It is the single source of truth shared by the pricing engine (item SuppliesCost
// = sum of its lines) and the API responses (each line's `total`), so a supply
// line's displayed total always reconciles with the item's supplies cost. There is
// exactly ONE math.Round per line so the per-line totals sum to the item cost
// without accumulating rounding drift.
//
// Guards: a non-positive packageSize is treated as 1 (so a malformed snapshot can
// never divide by zero or flip the sign); a non-positive quantity, product count or
// package price yields 0. The product is computed in float64 and clamped to
// [0, math.MaxInt64] before the int64 conversion: a float64 outside the int64 range
// converts to an implementation-defined value in Go, so even though the request
// validation bounds quantity_per_piece and product_quantity, an extreme package
// price is guarded here rather than silently wrapping to a negative/garbage cost.
func SupplyCostCents(quantityPerPiece float64, productQuantity int, packagePriceCents int64, packageSize float64) int64 {
	if packageSize <= 0 {
		packageSize = 1
	}
	if quantityPerPiece <= 0 || productQuantity <= 0 || packagePriceCents <= 0 {
		return 0
	}
	cents := math.Round(quantityPerPiece * float64(productQuantity) * float64(packagePriceCents) / packageSize)
	if cents >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if cents <= 0 {
		return 0
	}
	return int64(cents)
}
