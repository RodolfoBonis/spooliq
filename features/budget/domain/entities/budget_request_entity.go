package entities

import "github.com/google/uuid"

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

	// Filaments used in this item (1:N relationship)
	Filaments []BudgetItemFilamentRequest `json:"filaments" validate:"required,min=1,dive"`

	// Optional: specific cost preset for this item
	CostPresetID *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Optional: additional labor cost for this item (pintura, acabamento, etc)
	AdditionalLaborCost *int64 `json:"additional_labor_cost,omitempty" validate:"omitempty,gte=0"` // cents

	// Optional: notes specific to this item
	AdditionalNotes *string `json:"additional_notes,omitempty" validate:"omitempty,max=500"`

	// Order in the budget
	Order int `json:"order" validate:"gte=0"`
}

// CreateBudgetRequest represents the request to create a new budget
type CreateBudgetRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  uuid.UUID `json:"customer_id" validate:"required"`

	// Global presets (apply to all items unless overridden)
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`

	// Configuration flags
	IncludeEnergyCost bool `json:"include_energy_cost"`
	IncludeWasteCost  bool `json:"include_waste_cost"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// Items (products)
	Items []BudgetItemRequest `json:"items" validate:"required,min=1,dive"`
}

// UpdateBudgetRequest represents the request to update an existing budget
type UpdateBudgetRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  *uuid.UUID `json:"customer_id,omitempty"`

	// Global presets
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`

	// Configuration flags
	IncludeEnergyCost *bool `json:"include_energy_cost,omitempty"`
	IncludeWasteCost  *bool `json:"include_waste_cost,omitempty"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// Items (optional - if provided, replaces all items)
	Items *[]BudgetItemRequest `json:"items,omitempty" validate:"omitempty,min=1,dive"`
}

// UpdateStatusRequest represents the request to update budget status
type UpdateStatusRequest struct {
	Status BudgetStatus `json:"status" validate:"required,oneof=draft sent approved rejected printing completed"`
	Notes  string       `json:"notes,omitempty" validate:"omitempty,max=500"`
}
