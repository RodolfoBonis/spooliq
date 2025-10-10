package entities

import "time"

// CustomerInfo represents simplified customer information for budget responses
type CustomerInfo struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Document *string `json:"document,omitempty"`
}

// FilamentInfo represents simplified filament information for budget responses
type FilamentInfo struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	BrandName    string  `json:"brand_name"`
	MaterialName string  `json:"material_name"`
	Color        string  `json:"color"`
	PricePerKg   float64 `json:"price_per_kg"`
}

// PresetInfo represents simplified preset information
type PresetInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "machine", "energy", "cost"
}

// BudgetItemResponse represents a budget item with filament details
type BudgetItemResponse struct {
	ID          string        `json:"id"`
	BudgetID    string        `json:"budget_id"`
	FilamentID  string        `json:"filament_id"`
	Filament    *FilamentInfo `json:"filament"`
	Quantity    float64       `json:"quantity"`
	Order       int           `json:"order"`
	WasteAmount float64       `json:"waste_amount"`
	ItemCost    int64         `json:"item_cost"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// BudgetResponse represents the response for a single budget
type BudgetResponse struct {
	Budget        *BudgetEntity               `json:"budget"`
	Customer      *CustomerInfo               `json:"customer"`
	Items         []BudgetItemResponse        `json:"items"`
	MachinePreset *PresetInfo                 `json:"machine_preset,omitempty"`
	EnergyPreset  *PresetInfo                 `json:"energy_preset,omitempty"`
	CostPreset    *PresetInfo                 `json:"cost_preset,omitempty"`
	StatusHistory []BudgetStatusHistoryEntity `json:"status_history,omitempty"`
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
