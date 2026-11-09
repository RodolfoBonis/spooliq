package entities

import "math"

// SupplyUnits is the set of valid measurement units a supply may use. It is the
// single source of truth behind the `oneof` validation tag below.
var SupplyUnits = []string{"un", "par", "ml", "g", "m", "cm", "kg", "l"}

// UpsertSupplyRequestEntity represents the request payload for creating or
// updating a supply.
//
// A supply is priced as a PACKAGE: PackageSize units bought for PackagePrice
// (CENTS) — e.g. a 400 ml lotion (package_size 400, package_price 6990). Unit must
// be one of SupplyUnits.
//
// PackageSize and PackagePrice are pointers for backward compatibility with clients
// that still send only the legacy per-unit `unit_cost`: when BOTH are absent the
// use case falls back to package_size = 1 and package_price = unit_cost, so an old
// payload keeps pricing exactly as before. PackageSize must be > 0; PackagePrice
// (and the legacy UnitCost) must be >= 0.
//
// IsActive is a pointer so an absent value on update keeps the stored flag, while
// create defaults it to true when omitted (see the use case).
type UpsertSupplyRequestEntity struct {
	Name         string   `json:"name" validate:"required,min=1,max=255"`
	Unit         string   `json:"unit" validate:"required,oneof=un par ml g m cm kg l"`
	PackageSize  *float64 `json:"package_size,omitempty" validate:"omitempty,gt=0,lte=1000000"`
	PackagePrice *int64   `json:"package_price,omitempty" validate:"omitempty,gte=0"`
	// UnitCost is the legacy per-unit cost (CENTS). Kept only for backward
	// compatibility: it is used to derive the package fields when they are absent.
	// A pointer so "not sent" is distinguishable from an explicit 0.
	UnitCost    *int64  `json:"unit_cost,omitempty" validate:"omitempty,gte=0"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ResolvePackage resolves (packageSize, packagePrice) for a CREATE. When BOTH
// package fields are absent it falls back to package_size = 1 and package_price =
// the legacy unit_cost (0 when absent), so an old payload prices exactly as before.
// A missing size defaults to 1; a missing price uses the legacy unit_cost.
func (r *UpsertSupplyRequestEntity) ResolvePackage() (packageSize float64, packagePrice int64) {
	packageSize = 1
	if r.PackageSize != nil {
		packageSize = *r.PackageSize
	}
	switch {
	case r.PackagePrice != nil:
		packagePrice = *r.PackagePrice
	case r.UnitCost != nil:
		packagePrice = *r.UnitCost
	}
	return packageSize, packagePrice
}

// ResolvePackageUpdate resolves (packageSize, packagePrice) for an UPDATE of a
// supply currently stored as (storedSize, storedPrice). Fields the client does not
// send keep their stored values, so a partial PUT never zeroes the price.
//
// Legacy clients only know unit_cost: when no package field is sent and unit_cost
// equals the stored derived value, the package is kept as is; when unit_cost
// differs the client really changed the price, which is stored as a 1-unit
// package (the legacy meaning of unit_cost).
func (r *UpsertSupplyRequestEntity) ResolvePackageUpdate(storedSize float64, storedPrice int64) (packageSize float64, packagePrice int64) {
	if r.PackageSize == nil && r.PackagePrice == nil {
		if r.UnitCost != nil && *r.UnitCost != DeriveUnitCostCents(storedPrice, storedSize) {
			return 1, *r.UnitCost
		}
		return storedSize, storedPrice
	}
	packageSize, packagePrice = storedSize, storedPrice
	if r.PackageSize != nil {
		packageSize = *r.PackageSize
	}
	if r.PackagePrice != nil {
		packagePrice = *r.PackagePrice
	}
	return packageSize, packagePrice
}

// DeriveUnitCostCents is the per-unit cost (CENTS) implied by a package price and
// size: round(packagePriceCents / packageSize). It is the single source of truth
// for the stored supplies/budget_item_supplies unit_cost columns, so readers that
// predate package pricing still see a coherent per-unit value. A non-positive
// packageSize is treated as 1 and a non-positive price yields 0.
func DeriveUnitCostCents(packagePriceCents int64, packageSize float64) int64 {
	if packageSize <= 0 {
		packageSize = 1
	}
	if packagePriceCents <= 0 {
		return 0
	}
	cents := math.Round(float64(packagePriceCents) / packageSize)
	if cents >= float64(math.MaxInt64) {
		return math.MaxInt64
	}
	if cents <= 0 {
		return 0
	}
	return int64(cents)
}
