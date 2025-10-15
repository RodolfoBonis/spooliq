package entities

import "github.com/google/uuid"

// BudgetItemRequest represents a filament item in a budget request
type BudgetItemRequest struct {
	FilamentID uuid.UUID `json:"filament_id" validate:"required"`
	Quantity   float64   `json:"quantity" validate:"required,gt=0"` // grams
	Order      int       `json:"order" validate:"gte=0"`            // sequence for AMS

	// Product information (customer-facing)
	ProductName        string  `json:"product_name" validate:"required,min=1,max=255"`
	ProductDescription *string `json:"product_description,omitempty" validate:"omitempty,max=1000"`
	ProductQuantity    int     `json:"product_quantity" validate:"required,gt=0"`
	UnitPrice          int64   `json:"unit_price" validate:"required,gte=0"` // cents per unit
	ProductDimensions  *string `json:"product_dimensions,omitempty" validate:"omitempty,max=100"`
}

// CreateBudgetRequest represents the request to create a new budget
type CreateBudgetRequest struct {
	Name        string    `json:"name" validate:"required,min=1,max=255"`
	Description string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  uuid.UUID `json:"customer_id" validate:"required"`

	// Print time (manual input for now)
	PrintTimeHours   int `json:"print_time_hours" validate:"gte=0"`
	PrintTimeMinutes int `json:"print_time_minutes" validate:"gte=0,lt=60"`

	// Presets
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Configuration
	IncludeEnergyCost bool     `json:"include_energy_cost"`
	IncludeLaborCost  bool     `json:"include_labor_cost"`
	IncludeWasteCost  bool     `json:"include_waste_cost"`
	LaborCostPerHour  *float64 `json:"labor_cost_per_hour,omitempty" validate:"omitempty,gte=0"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// Items
	Items []BudgetItemRequest `json:"items" validate:"required,min=1,dive"`
}

// UpdateBudgetRequest represents the request to update an existing budget
type UpdateBudgetRequest struct {
	Name        *string    `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	CustomerID  *uuid.UUID `json:"customer_id,omitempty"`

	// Print time
	PrintTimeHours   *int `json:"print_time_hours,omitempty" validate:"omitempty,gte=0"`
	PrintTimeMinutes *int `json:"print_time_minutes,omitempty" validate:"omitempty,gte=0,lt=60"`

	// Presets
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	CostPresetID    *uuid.UUID `json:"cost_preset_id,omitempty"`

	// Configuration
	IncludeEnergyCost *bool    `json:"include_energy_cost,omitempty"`
	IncludeLaborCost  *bool    `json:"include_labor_cost,omitempty"`
	IncludeWasteCost  *bool    `json:"include_waste_cost,omitempty"`
	LaborCostPerHour  *float64 `json:"labor_cost_per_hour,omitempty" validate:"omitempty,gte=0"`

	// Additional fields for PDF
	DeliveryDays *int    `json:"delivery_days,omitempty" validate:"omitempty,gte=0"`
	PaymentTerms *string `json:"payment_terms,omitempty" validate:"omitempty,max=1000"`
	Notes        *string `json:"notes,omitempty" validate:"omitempty,max=2000"`

	// Items (optional - if provided, replaces all items)
	Items *[]BudgetItemRequest `json:"items,omitempty" validate:"omitempty,min=1,dive"`
}

// UpdateStatusRequest represents the request to update budget status
type UpdateStatusRequest struct {
	NewStatus BudgetStatus `json:"new_status" validate:"required,oneof=draft sent approved rejected printing completed"`
	Notes     string       `json:"notes,omitempty" validate:"omitempty,max=500"`
}
