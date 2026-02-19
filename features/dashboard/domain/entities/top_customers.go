package entities

// TopCustomer represents a top customer by revenue.
type TopCustomer struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email,omitempty"`
	TotalRevenue int64  `json:"total_revenue"`
	BudgetCount  int    `json:"budget_count"`
	AvgTicket    int64  `json:"avg_ticket"`
}

// TopCustomersResponse contains top customers data.
type TopCustomersResponse struct {
	Customers []TopCustomer `json:"customers"`
	Period    string        `json:"period"`
}
