package entities

// RevenueTrendPoint represents a data point in the revenue trend.
type RevenueTrendPoint struct {
	Date        string `json:"date"`
	Revenue     int64  `json:"revenue"`
	Cost        int64  `json:"cost"`
	Profit      int64  `json:"profit"`
	BudgetCount int    `json:"budget_count"`
}

// RevenueTrendResponse contains revenue trend data points.
type RevenueTrendResponse struct {
	Points []RevenueTrendPoint `json:"points"`
	Period string              `json:"period"`
}
