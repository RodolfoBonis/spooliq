package entities

import (
	"encoding/json"
	"time"
)

// CustomerInfo represents simplified customer information for budget responses
type CustomerInfo struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Document *string `json:"document,omitempty"`
}

// FilamentInfo represents simplified filament information for budget responses (legacy, kept for compatibility)
type FilamentInfo struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	BrandName    string  `json:"brand_name"`
	MaterialName string  `json:"material_name"`
	Color        string  `json:"color"`
	PricePerKg   float64 `json:"price_per_kg"`
}

// FilamentUsageInfo represents detailed filament usage for a budget item
type FilamentUsageInfo struct {
	FilamentID   string `json:"filament_id"`
	FilamentName string `json:"filament_name"`
	BrandName    string `json:"brand_name"`
	MaterialName string `json:"material_name"`

	// Legacy color field (maintained for backward compatibility)
	Color string `json:"color"`

	// Advanced color system
	ColorType    string          `json:"color_type"`
	ColorData    json.RawMessage `json:"color_data,omitempty"`
	ColorHex     string          `json:"color_hex,omitempty"`
	ColorPreview string          `json:"color_preview,omitempty"`

	Quantity float64 `json:"quantity"` // gramas TOTAL para este item
	Cost     int64   `json:"cost"`     // centavos
	Order    int     `json:"order"`
}

// PresetInfo represents simplified preset information
type PresetInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "machine", "energy", "cost"
}

// BudgetItemResponse represents a budget item (product) with all filaments and costs
type BudgetItemResponse struct {
	ID       string `json:"id"`
	BudgetID string `json:"budget_id"`

	// Product information (customer-facing)
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"`
	ProductDimensions  *string `json:"product_dimensions,omitempty"`

	// Print time for this item
	PrintTimeHours   int    `json:"print_time_hours"`
	PrintTimeMinutes int    `json:"print_time_minutes"`
	PrintTimeDisplay string `json:"print_time_display"` // "5h30m"

	// Cost preset and additional costs
	CostPresetID        *string `json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64  `json:"additional_labor_cost,omitempty"` // cents
	AdditionalNotes     *string `json:"additional_notes,omitempty"`

	// Calculated costs for this item
	FilamentCost  int64 `json:"filament_cost"`   // cents
	WasteCost     int64 `json:"waste_cost"`      // cents
	EnergyCost    int64 `json:"energy_cost"`     // cents
	LaborCost     int64 `json:"labor_cost"`      // cents
	ItemTotalCost int64 `json:"item_total_cost"` // cents (sum of all)
	UnitPrice     int64 `json:"unit_price"`      // cents per unit

	// Filaments used in this item
	Filaments []FilamentUsageInfo `json:"filaments"`

	Order int `json:"order"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BudgetResponse represents the response for a single budget
type BudgetResponse struct {
	// Embed all BudgetEntity fields directly
	*BudgetEntity

	Customer      *CustomerInfo               `json:"customer"`
	Items         []BudgetItemResponse        `json:"items"`
	MachinePreset *PresetInfo                 `json:"machine_preset,omitempty"`
	EnergyPreset  *PresetInfo                 `json:"energy_preset,omitempty"`
	StatusHistory []BudgetStatusHistoryEntity `json:"status_history,omitempty"`

	// Total print time (sum of all items)
	TotalPrintTimeHours   int    `json:"total_print_time_hours"`
	TotalPrintTimeMinutes int    `json:"total_print_time_minutes"`
	TotalPrintTimeDisplay string `json:"total_print_time_display"` // "14h15m"
}

// ListBudgetsResponse represents the response for listing budgets
type ListBudgetsResponse struct {
	Data       []BudgetResponse `json:"data"`
	Total      int              `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// BudgetCalculationResponse represents the response for budget calculation
type BudgetCalculationResponse struct {
	BudgetID       string  `json:"budget_id"`
	FilamentCost   float64 `json:"filament_cost"` // in currency
	WasteCost      float64 `json:"waste_cost"`
	EnergyCost     float64 `json:"energy_cost"`
	LaborCost      float64 `json:"labor_cost"`
	TotalCost      float64 `json:"total_cost"`
	ItemsBreakdown []struct {
		FilamentID   string  `json:"filament_id"`
		FilamentName string  `json:"filament_name"`
		Quantity     float64 `json:"quantity"`     // grams
		WasteAmount  float64 `json:"waste_amount"` // grams
		ItemCost     float64 `json:"item_cost"`
	} `json:"items_breakdown"`
}

// CompanyInfo represents simplified company information for PDF generation
type CompanyInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	WhatsApp  *string `json:"whatsapp,omitempty"`
	Instagram *string `json:"instagram,omitempty"`
	Website   *string `json:"website,omitempty"`
	LogoURL   *string `json:"logo_url,omitempty"`
}
