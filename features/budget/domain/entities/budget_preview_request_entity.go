package entities

import "github.com/google/uuid"

// PreviewBudgetRequest is the body for POST /v1/budgets/preview. It mirrors
// CreateBudgetRequest but the customer is OPTIONAL (a preview is a stateless cost
// estimate, not tied to a saved customer). Items are still required. Nothing is
// persisted when this request is processed.
type PreviewBudgetRequest struct {
	Name        string     `json:"name,omitempty" validate:"omitempty,max=255"`
	Description string     `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  *uuid.UUID `json:"customer_id,omitempty"`

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

	// Discount (optional). DiscountType ∈ {"percent","fixed"}.
	DiscountType  *string  `json:"discount_type,omitempty" validate:"omitempty,oneof=percent fixed"`
	DiscountValue *float64 `json:"discount_value,omitempty" validate:"omitempty,gte=0"`

	// Shipping (optional). ShippingOverride is in cents.
	IncludeShipping  bool   `json:"include_shipping"`
	ShippingOverride *int64 `json:"shipping_override,omitempty" validate:"omitempty,gte=0"`

	// TaxRate (optional, percent 0..<100). When omitted the company default is used.
	TaxRate *float64 `json:"tax_rate,omitempty" validate:"omitempty,gte=0,lt=100"`

	// Additional fields (echoed back on the preview response for convenience)
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// Items (products) - required
	Items []BudgetItemRequest `json:"items" validate:"required,min=1,dive"`
}
