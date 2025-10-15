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

	// Print time for THIS item (not global)
	PrintTimeHours   int `json:"print_time_hours"`
	PrintTimeMinutes int `json:"print_time_minutes"`

	// Additional costs specific to this item
	CostPresetID        *uuid.UUID `json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64     `json:"additional_labor_cost,omitempty"` // cents
	AdditionalNotes     *string    `json:"additional_notes,omitempty"`

	// Calculated costs per item
	FilamentCost  int64 `json:"filament_cost"`   // cents
	WasteCost     int64 `json:"waste_cost"`      // cents
	EnergyCost    int64 `json:"energy_cost"`     // cents
	LaborCost     int64 `json:"labor_cost"`      // cents
	ItemTotalCost int64 `json:"item_total_cost"` // cents (sum of all costs)

	// OLD FIELDS (deprecated, kept for compatibility)
	WasteAmount float64 `json:"waste_amount"` // grams (deprecated)
	ItemCost    int64   `json:"item_cost"`    // cents (deprecated, use ItemTotalCost)

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
