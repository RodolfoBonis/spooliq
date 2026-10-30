package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetItemSupplyEntity represents a non-printed supply (insumo) attached to a
// budget item. Name, Unit and UnitCost are a SNAPSHOT taken from the supply
// catalog at create/update/recalculate time, so the budget keeps reproducible
// values even if the catalog supply is later changed or deleted. QuantityPerPiece
// is how many units of the supply each printed piece consumes.
type BudgetItemSupplyEntity struct {
	ID               uuid.UUID `json:"id"`
	BudgetItemID     uuid.UUID `json:"budget_item_id"`
	SupplyID         uuid.UUID `json:"supply_id"`
	OrganizationID   string    `json:"organization_id"`
	QuantityPerPiece float64   `json:"quantity_per_piece"`
	Order            int       `json:"order"`

	// Snapshot of the catalog supply at the last calculation.
	Name     string `json:"name"`
	Unit     string `json:"unit"`
	UnitCost int64  `json:"unit_cost"` // cents

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SupplySnapshot is the subset of a catalog supply snapshotted onto a budget item
// supply (and used to price a preview). UnitCost is in cents.
type SupplySnapshot struct {
	Name     string
	Unit     string
	UnitCost int64
}
