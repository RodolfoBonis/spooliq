package entities

// GoalMetric identifies what a monthly goal measures.
type GoalMetric string

// Goal metrics. Money targets are in cents; approval rate is a percentage.
const (
	GoalRevenue      GoalMetric = "revenue"       // net revenue of sales approved in the month
	GoalProfit       GoalMetric = "profit"        // profit of sales approved in the month
	GoalBudgets      GoalMetric = "budgets"       // number of sales approved in the month
	GoalApprovalRate GoalMetric = "approval_rate" // approved / decided in the month
)

// GoalMetrics lists every metric in display order.
var GoalMetrics = []GoalMetric{GoalRevenue, GoalProfit, GoalBudgets, GoalApprovalRate}

// Valid reports whether m is a known goal metric.
func (m GoalMetric) Valid() bool {
	for _, known := range GoalMetrics {
		if m == known {
			return true
		}
	}
	return false
}

// Cumulative reports whether the metric accumulates through the month (and can
// therefore be projected to month end). Approval rate is a ratio, not a sum.
func (m GoalMetric) Cumulative() bool { return m != GoalApprovalRate }

// GoalTarget is a user-defined monthly target for one metric.
type GoalTarget struct {
	Metric GoalMetric `json:"metric" binding:"required"`
	Target float64    `json:"target" binding:"gte=0"`
}

// GoalTargetsResponse lists the organization's configured monthly targets.
type GoalTargetsResponse struct {
	Goals []GoalTarget `json:"goals"`
}

// SaveGoalTargetsRequest replaces the targets for the given metrics. A target of
// zero removes the goal for that metric.
type SaveGoalTargetsRequest struct {
	Goals []GoalTarget `json:"goals" binding:"required,dive"`
}
