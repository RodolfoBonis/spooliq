package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetItemFilamentRequest represents a filament in a budget item request
type BudgetItemFilamentRequest struct {
	FilamentID uuid.UUID `json:"filament_id" validate:"required"`
	Quantity   float64   `json:"quantity" validate:"required,gt=0"` // gramas TOTAL
	Order      int       `json:"order" validate:"gte=1"`
}

// BudgetItemRequest represents a product item in a budget request
type BudgetItemRequest struct {
	// Product information (customer-facing)
	ProductName        string  `json:"product_name" validate:"required,min=1,max=255"`
	ProductDescription *string `json:"product_description,omitempty" validate:"omitempty,max=1000"`
	ProductQuantity    int     `json:"product_quantity" validate:"required,gt=0"` // number of units
	ProductDimensions  *string `json:"product_dimensions,omitempty" validate:"omitempty,max=100"`

	// Print time for THIS item
	PrintTimeHours   int `json:"print_time_hours" validate:"gte=0"`
	PrintTimeMinutes int `json:"print_time_minutes" validate:"gte=0,lt=60"`

	// Labor breakdown for this item
	SetupTimeMinutes        int `json:"setup_time_minutes" validate:"gte=0"`         // Setup time for this product (minutes)
	ManualLaborMinutesTotal int `json:"manual_labor_minutes_total" validate:"gte=0"` // Total manual labor time for ALL units (minutes)
	PostProcessingMinutes   int `json:"post_processing_minutes" validate:"gte=0"`    // Post-processing time for this item (minutes)
	SupportRemovalMinutes   int `json:"support_removal_minutes" validate:"gte=0"`    // Support-removal time for this item (minutes)

	// Filaments used in this item (1:N relationship)
	Filaments []BudgetItemFilamentRequest `json:"filaments" validate:"required,min=1,dive"`

	// Optional: specific cost preset for this item. When omitted, the budget-level
	// cost preset (resolved from the request/profile/org defaults) is used for this
	// item's setup/labor/post-processing/packaging/QC/failure rates.
	CostPresetID *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Optional: notes specific to this item
	AdditionalNotes *string `json:"additional_notes,omitempty" validate:"omitempty,max=500"`

	// Optional: link this item to a 3D model (must belong to the organization)
	Model3DID *uuid.UUID `json:"model_3d_id,omitempty"`

	// Order in the budget
	Order int `json:"order" validate:"gte=0"`
}

// CreateBudgetRequest represents the request to create a new budget
type CreateBudgetRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  uuid.UUID `json:"customer_id" validate:"required"`

	// Optional print profile. When provided it supplies the machine/energy/cost
	// presets for any slot not explicitly set below (see the budget preset resolver).
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`

	// Global presets (apply to all items unless overridden). An explicit value here
	// always wins over the profile and org defaults for that slot.
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	// CostPresetID is the budget-level cost preset driving overhead/profit (and the
	// setup/labor rate fallback for items without their own cost preset).
	CostPresetID *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Configuration flags. IncludeMachineCost defaults to TRUE when omitted.
	IncludeEnergyCost  bool  `json:"include_energy_cost"`
	IncludeWasteCost   bool  `json:"include_waste_cost"`
	IncludeMachineCost *bool `json:"include_machine_cost,omitempty"`

	// Discount (optional). DiscountType ∈ {"percent","fixed"}; DiscountValue is a
	// 0-100 percent for "percent" or a reais amount (>= 0) for "fixed".
	DiscountType  *string  `json:"discount_type,omitempty" validate:"omitempty,oneof=percent fixed"`
	DiscountValue *float64 `json:"discount_value,omitempty" validate:"omitempty,gte=0"`

	// Shipping (optional). ShippingOverride is in cents.
	IncludeShipping  bool   `json:"include_shipping"`
	ShippingOverride *int64 `json:"shipping_override,omitempty" validate:"omitempty,gte=0"`

	// TaxRate (optional, percent 0..<100). When omitted the company default is used.
	TaxRate *float64 `json:"tax_rate,omitempty" validate:"omitempty,gte=0,lt=100"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// ValidUntil is the optional quote validity date (date-only semantics; stored at
	// end of day in America/Sao_Paulo). When omitted it is computed on first send.
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	// Items (products)
	Items []BudgetItemRequest `json:"items" validate:"required,min=1,dive"`
}

// UpdateBudgetRequest represents the request to update an existing budget
type UpdateBudgetRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  *uuid.UUID `json:"customer_id,omitempty"`

	// Optional print profile. Presets are ONLY re-resolved from the profile/org
	// defaults when profile_id is explicitly provided in the update; otherwise the
	// stored preset values are kept (partial-update semantics).
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`

	// Global presets
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	// CostPresetID is the budget-level cost preset driving overhead/profit.
	CostPresetID *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Configuration flags
	IncludeEnergyCost  *bool `json:"include_energy_cost,omitempty"`
	IncludeWasteCost   *bool `json:"include_waste_cost,omitempty"`
	IncludeMachineCost *bool `json:"include_machine_cost,omitempty"`

	// Discount (optional).
	DiscountType  *string  `json:"discount_type,omitempty" validate:"omitempty,oneof=percent fixed"`
	DiscountValue *float64 `json:"discount_value,omitempty" validate:"omitempty,gte=0"`

	// Shipping (optional). ShippingOverride is in cents.
	IncludeShipping  *bool  `json:"include_shipping,omitempty"`
	ShippingOverride *int64 `json:"shipping_override,omitempty" validate:"omitempty,gte=0"`

	// TaxRate (optional, percent 0..<100).
	TaxRate *float64 `json:"tax_rate,omitempty" validate:"omitempty,gte=0,lt=100"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// ValidUntil is the optional quote validity date (date-only; stored end of day,
	// America/Sao_Paulo). Settable on update.
	ValidUntil *time.Time `json:"valid_until,omitempty"`

	// Items (optional - if provided, replaces all items)
	Items *[]BudgetItemRequest `json:"items,omitempty" validate:"omitempty,min=1,dive"`
}

// UpdateStatusRequest represents the request to update budget status
type UpdateStatusRequest struct {
	Status BudgetStatus `json:"status" validate:"required,oneof=draft sent approved rejected printing completed expired cancelled"`
	Notes  string       `json:"notes,omitempty" validate:"omitempty,max=500"`
}
