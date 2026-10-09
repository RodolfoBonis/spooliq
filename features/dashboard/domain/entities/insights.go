package entities

// InsightSeverity orders insights: critical first, positive last.
type InsightSeverity string

// Insight severities.
const (
	SeverityCritical InsightSeverity = "critical"
	SeverityWarning  InsightSeverity = "warning"
	SeverityInfo     InsightSeverity = "info"
	SeverityPositive InsightSeverity = "positive"
)

// Rank returns the sort rank of the severity (lower comes first).
func (s InsightSeverity) Rank() int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

// InsightAction points the client to where the user can act. Target is a
// client-agnostic destination (budgets, budget, customers, customer, filaments,
// filament, materials, machines, costs, goals); ID and Filter refine it.
type InsightAction struct {
	Label  string `json:"label"`
	Target string `json:"target"`
	ID     string `json:"id,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// Insight is a rule-generated, actionable observation about the business.
type Insight struct {
	Kind     string          `json:"kind"`
	Severity InsightSeverity `json:"severity"`
	Title    string          `json:"title"`
	Detail   string          `json:"detail"`
	// Metric is the key number already formatted in pt-BR ("12,4%", "R$ 350,00").
	Metric string `json:"metric,omitempty"`
	// Impact is the estimated money at stake in cents, used for ranking.
	Impact int64          `json:"impact"`
	Action *InsightAction `json:"action,omitempty"`
}

// InsightsResponse lists the most relevant insights for the period.
type InsightsResponse struct {
	Insights []Insight `json:"insights"`
	Period   string    `json:"period"`
}

// NamedRef identifies an entity by ID and display name.
type NamedRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StockShortfall is a stock-tracked filament whose balance does not cover the
// last 30 days of consumption.
type StockShortfall struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	StockGrams      int64   `json:"stock_grams"`
	ConsumedGrams30 float64 `json:"consumed_grams_30d"`
}

// InsightSignals are the extra aggregates the insight rules need beyond the
// overview, profitability, response times and goals.
type InsightSignals struct {
	StaleDrafts int
	// InactiveRepeatCustomers bought 2+ times but had no budget in 60 days.
	InactiveRepeatCustomers      []NamedRef
	InactiveRepeatCustomersTotal int
	// Waste vs filament cost of the period's sales, in cents.
	WasteCost      int64
	FilamentCost   int64
	StockShortfall []StockShortfall
}
