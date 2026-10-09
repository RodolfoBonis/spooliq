package entities

// Goal represents a monthly goal with progress and month-end projection.
//
// Goals are user-defined (see GoalTarget). Metrics without a target are still
// returned with Configured=false so clients can offer "definir meta".
type Goal struct {
	Metric     GoalMetric `json:"metric"`
	Name       string     `json:"name"`
	Configured bool       `json:"configured"`
	Current    float64    `json:"current"`
	Target     float64    `json:"target"`
	Progress   float64    `json:"progress"` // current / target * 100, capped at 100
	Unit       string     `json:"unit"`     // cents | count | percent
	// Projected is the month-end value at the current daily pace (cumulative
	// metrics) or the current value (approval rate).
	Projected         float64 `json:"projected"`
	ProjectedProgress float64 `json:"projected_progress"` // projected / target * 100 (not capped)
	// RequiredPerDay is what is still needed per remaining day to hit the target
	// (zero when already reached or for approval rate).
	RequiredPerDay float64 `json:"required_per_day"`
	DaysLeft       int     `json:"days_left"`
}

// Alert represents a dashboard alert notification.
type Alert struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Count      int    `json:"count,omitempty"`
	EntityType string `json:"entity_type,omitempty"`
}

// GoalsAlertsResponse contains goals progress and active alerts.
type GoalsAlertsResponse struct {
	Goals  []Goal  `json:"goals"`
	Alerts []Alert `json:"alerts"`
	// Month is the goals month in America/Sao_Paulo, formatted YYYY-MM.
	Month string `json:"month"`
}
