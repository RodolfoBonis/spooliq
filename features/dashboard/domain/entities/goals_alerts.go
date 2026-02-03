package entities

// Goal represents a dashboard goal with progress tracking.
type Goal struct {
	Name     string  `json:"name"`
	Current  float64 `json:"current"`
	Target   float64 `json:"target"`
	Progress float64 `json:"progress"`
	Unit     string  `json:"unit"`
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
}
