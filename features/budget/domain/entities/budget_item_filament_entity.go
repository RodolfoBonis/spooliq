package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetItemFilamentEntity represents a filament used in a budget item (multi-filament support)
type BudgetItemFilamentEntity struct {
	ID             uuid.UUID `json:"id"`
	BudgetItemID   uuid.UUID `json:"budget_item_id"`
	FilamentID     uuid.UUID `json:"filament_id"`
	OrganizationID string    `json:"organization_id"`
	Quantity       float64   `json:"quantity"` // gramas TOTAL para este item
	Order          int       `json:"order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
