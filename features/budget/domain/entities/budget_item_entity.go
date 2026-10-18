package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetItemEntity represents a filament item in a budget
type BudgetItemEntity struct {
	ID             uuid.UUID `json:"id"`
	BudgetID       uuid.UUID `json:"budget_id"`
	FilamentID     uuid.UUID `json:"filament_id"`
	OrganizationID string    `json:"organization_id"`

	// Filament quantity (internal - for cost calculation)
	Quantity float64 `json:"quantity"` // grams
	Order    int     `json:"order"`    // sequence for color changes (1, 2, 3...)

	// Product information (customer-facing - for PDF and quotes)
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"` // number of units
	// UnitsPerPrint is how many units one run of the sliced file produces. When
	// set, the client scaled print time, grams and labor from one run to the
	// quantity; the stored values are still totals for the item.
	UnitsPerPrint *int `json:"units_per_print,omitempty"`
	// PerPrint keeps the client\'s values for ONE run of the file, so the item
	// can be re-edited without rounding drift. Opaque to pricing.
	PerPrint          *PerPrintValues `json:"per_print,omitempty"`
	UnitPrice         int64           `json:"unit_price"` // cents per unit
	ProductDimensions *string         `json:"product_dimensions,omitempty"`

	// Print time for THIS item (not global)
	PrintTimeHours   int `json:"print_time_hours"`
	PrintTimeMinutes int `json:"print_time_minutes"`

	// Labor breakdown fields
	SetupTimeMinutes        int `json:"setup_time_minutes"`         // Setup time for this product (minutes)
	ManualLaborMinutesTotal int `json:"manual_labor_minutes_total"` // Total manual labor time for ALL units (minutes)
	PostProcessingMinutes   int `json:"post_processing_minutes"`    // Post-processing time for this item (minutes)
	SupportRemovalMinutes   int `json:"support_removal_minutes"`    // Support-removal time for this item (minutes)

	// Additional costs specific to this item
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"`
	AdditionalNotes *string    `json:"additional_notes,omitempty"`

	// Optional link to a 3D model
	Model3DID *uuid.UUID `json:"model_3d_id,omitempty"`

	// Calculated costs per item
	FilamentCost       int64 `json:"filament_cost"`        // cents
	WasteCost          int64 `json:"waste_cost"`           // cents
	EnergyCost         int64 `json:"energy_cost"`          // cents
	MachineCost        int64 `json:"machine_cost"`         // cents
	SetupCost          int64 `json:"setup_cost"`           // cents
	ManualLaborCost    int64 `json:"manual_labor_cost"`    // cents
	PostProcessingCost int64 `json:"post_processing_cost"` // cents
	SupportRemovalCost int64 `json:"support_removal_cost"` // cents
	PackagingCost      int64 `json:"packaging_cost"`       // cents
	QualityControlCost int64 `json:"quality_control_cost"` // cents
	FailureCost        int64 `json:"failure_cost"`         // cents
	ItemTotalCost      int64 `json:"item_total_cost"`      // cents (sum of all costs)

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PerPrintValues are an item's inputs for one run of a sliced file that
// produces UnitsPerPrint units. Pricing uses the scaled totals instead.
type PerPrintValues struct {
	PrintMinutes          int       `json:"print_minutes" validate:"gte=0"`
	SetupMinutes          int       `json:"setup_minutes" validate:"gte=0"`
	LaborMinutes          int       `json:"labor_minutes" validate:"gte=0"`
	PostProcessingMinutes int       `json:"post_processing_minutes" validate:"gte=0"`
	SupportRemovalMinutes int       `json:"support_removal_minutes" validate:"gte=0"`
	FilamentGrams         []float64 `json:"filament_grams" validate:"dive,gte=0"` // in filament order
}
