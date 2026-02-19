package entities

// BudgetStatusCount represents a count of budgets by status.
type BudgetStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// OverviewResponse contains dashboard overview metrics.
type OverviewResponse struct {
	TotalRevenue       int64               `json:"total_revenue"`
	RevenueChange      float64             `json:"revenue_change"`
	TotalBudgets       int                 `json:"total_budgets"`
	BudgetsChange      float64             `json:"budgets_change"`
	AvgTicket          int64               `json:"avg_ticket"`
	AvgTicketChange    float64             `json:"avg_ticket_change"`
	ApprovalRate       float64             `json:"approval_rate"`
	ApprovalRateChange float64             `json:"approval_rate_change"`
	AvgProfitMargin    float64             `json:"avg_profit_margin"`
	ProfitMarginChange float64             `json:"profit_margin_change"`
	NewCustomers       int                 `json:"new_customers"`
	NewCustomersChange float64             `json:"new_customers_change"`
	BudgetsByStatus    []BudgetStatusCount `json:"budgets_by_status"`
	Period             string              `json:"period"`
}
