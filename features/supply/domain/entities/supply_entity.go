package entities

import (
	"time"

	"github.com/google/uuid"
)

// SupplyEntity represents a non-printed supply (insumo) in the domain layer.
//
// A supply is a per-piece consumable that is NOT a filament and has no stock
// control — e.g. an LED, a dab of hydrating cream or a keyring. It carries a
// unit cost (in CENTS) that, when attached to a budget item, is treated as a
// direct cost that receives the same overhead + margin as every other cost.
//
// UnitCost serializes WITHOUT omitempty so a legitimate 0 is preserved in the
// payload rather than silently dropped.
type SupplyEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Unit           string     `json:"unit"`      // one of un, par, ml, g, m, cm, kg, l
	UnitCost       int64      `json:"unit_cost"` // cents, >= 0
	Description    *string    `json:"description,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
