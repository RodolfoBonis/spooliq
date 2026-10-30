package entities

import (
	"time"

	"github.com/google/uuid"
)

// SupplyEntity represents a non-printed supply (insumo) in the domain layer.
//
// A supply is a per-piece consumable that is NOT a filament and has no stock
// control — e.g. an LED, a dab of hydrating cream or a keyring. It is registered as
// a PACKAGE: PackageSize units of its measurement unit bought for PackagePrice
// (CENTS) — e.g. a 400 ml lotion for R$69,90. When attached to a budget item it is
// treated as a direct cost that receives the same overhead + margin as every other
// cost (see pricing.SupplyCostCents).
//
// UnitCost is DERIVED from the package (round(PackagePrice / PackageSize)) and kept
// stored so readers predating package pricing still see a coherent per-unit value.
// PackageSize, PackagePrice and UnitCost serialize WITHOUT omitempty so a legitimate
// 0 is preserved in the payload rather than silently dropped.
type SupplyEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Unit           string     `json:"unit"`          // one of un, par, ml, g, m, cm, kg, l
	PackageSize    float64    `json:"package_size"`  // units per package, > 0
	PackagePrice   int64      `json:"package_price"` // cents per package, >= 0
	UnitCost       int64      `json:"unit_cost"`     // cents, DERIVED = round(package_price / package_size)
	Description    *string    `json:"description,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
