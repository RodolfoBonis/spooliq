package entities

// CustomerResponse represents the response for a single customer
type CustomerResponse struct {
	Customer     *CustomerEntity `json:"customer"`
	BudgetCount  int             `json:"budget_count,omitempty"`
	TotalBudgets *int64          `json:"total_budgets,omitempty"` // Total em centavos
	Budgets      []BudgetSummary `json:"budgets,omitempty"`
}

// ListCustomersResponse represents the response for listing customers
type ListCustomersResponse struct {
	Data       []CustomerResponse `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}
