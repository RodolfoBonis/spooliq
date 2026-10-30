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

	// Profit metrics: sales (approved, printing, completed) counted by approval
	// date. Money in cents. NetRevenue excludes tax and shipping; Profit is
	// profit_amount minus discount; ProfitRealized + ProfitForecast = Profit.
	NetRevenue               int64   `json:"net_revenue"`
	NetRevenueChange         float64 `json:"net_revenue_change"`
	Profit                   int64   `json:"profit"`
	ProfitChange             float64 `json:"profit_change"`
	ProfitRealized           int64   `json:"profit_realized"`
	ProfitForecast           int64   `json:"profit_forecast"`
	ProductionCost           int64   `json:"production_cost"`
	ProfitMargin             float64 `json:"profit_margin"` // weighted: profit / net revenue * 100
	ProfitMarginPointsChange float64 `json:"profit_margin_points_change"`
	ProfitPerPrintHour       int64   `json:"profit_per_print_hour"`
	ProfitPerPrintHourChange float64 `json:"profit_per_print_hour_change"`
	PrintHours               float64 `json:"print_hours"`
	SalesCount               int     `json:"sales_count"`
	ApprovalRatePointsChange float64 `json:"approval_rate_points_change"`
}
