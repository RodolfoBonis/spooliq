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

	// Filament quantity (internal - for cost calculation)
	Quantity float64 `json:"quantity"` // grams
	Order    int     `json:"order"`    // sequence for color changes (1, 2, 3...)

	// Product information (customer-facing - for PDF and quotes)
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"` // number of units
	UnitPrice          int64   `json:"unit_price"`       // cents per unit
	ProductDimensions  *string `json:"product_dimensions,omitempty"`

	// Calculated values
	WasteAmount float64 `json:"waste_amount"` // grams (for AMS color changes)
	ItemCost    int64   `json:"item_cost"`    // cents (total cost for this item)

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
