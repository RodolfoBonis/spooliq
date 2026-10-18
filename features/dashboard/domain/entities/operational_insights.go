package entities

// CostBreakdown represents the percentage breakdown of costs.
type CostBreakdown struct {
	FilamentPct float64 `json:"filament_pct"`
	WastePct    float64 `json:"waste_pct"`
	EnergyPct   float64 `json:"energy_pct"`
	SetupPct    float64 `json:"setup_pct"`
	LaborPct    float64 `json:"labor_pct"`
	OverheadPct float64 `json:"overhead_pct"`
	// Components previously left out of the breakdown (additive fields).
	MachinePct        float64 `json:"machine_pct"`
	PostProcessingPct float64 `json:"post_processing_pct"`
	PackagingPct      float64 `json:"packaging_pct"`
	QualityControlPct float64 `json:"quality_control_pct"`
	FailurePct        float64 `json:"failure_pct"`
}

// OperationalInsightsResponse contains operational metrics.
type OperationalInsightsResponse struct {
	AvgTicket           int64         `json:"avg_ticket"`
	AvgTicketChange     float64       `json:"avg_ticket_change"`
	AvgProfitMargin     float64       `json:"avg_profit_margin"`
	ProfitMarginChange  float64       `json:"profit_margin_change"`
	TotalPrintTimeHours float64       `json:"total_print_time_hours"`
	PrintTimeChange     float64       `json:"print_time_change"`
	RejectionRate       float64       `json:"rejection_rate"`
	RejectionRateChange float64       `json:"rejection_rate_change"`
	CostBreakdown       CostBreakdown `json:"cost_breakdown"`
	Period              string        `json:"period"`
}
