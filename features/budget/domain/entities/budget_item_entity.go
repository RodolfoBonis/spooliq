package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetItemEntity represents a filament item in a budget
type BudgetItemEntity struct {
	ID         uuid.UUID `json:"id"`
	BudgetID   uuid.UUID `json:"budget_id"`
	FilamentID uuid.UUID `json:"filament_id"`

	// Quantity and order (for AMS color changes)
	Quantity float64 `json:"quantity"` // grams
	Order    int     `json:"order"`    // sequence for color changes (1, 2, 3...)

	// Calculated values
	WasteAmount float64 `json:"waste_amount"` // grams (for AMS color changes)
	ItemCost    int64   `json:"item_cost"`    // cents

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
