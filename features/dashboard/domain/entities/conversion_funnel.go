package entities

// FunnelStep represents a step in the conversion funnel.
type FunnelStep struct {
	Status         string  `json:"status"`
	Count          int     `json:"count"`
	ConversionRate float64 `json:"conversion_rate"`
}

// ConversionFunnelResponse contains conversion funnel data.
type ConversionFunnelResponse struct {
	Steps             []FunnelStep `json:"steps"`
	TotalBudgets      int          `json:"total_budgets"`
	OverallConversion float64      `json:"overall_conversion"`
	Period            string       `json:"period"`
}
