package entities

import "time"

// ResponseTimesResponse describes how customers respond to sent budgets.
// Durations are in hours, measured from the first "sent" to the decision.
type ResponseTimesResponse struct {
	ApprovalMedianHours  float64 `json:"approval_median_hours"`
	ApprovalP75Hours     float64 `json:"approval_p75_hours"`
	RejectionMedianHours float64 `json:"rejection_median_hours"`
	Approved             int     `json:"approved"`
	Rejected             int     `json:"rejected"`
	Expired              int     `json:"expired"`
	ExpirationRate       float64 `json:"expiration_rate"` // expired / decided * 100
	RejectionRate        float64 `json:"rejection_rate"`  // rejected / decided * 100
	// ApprovalBuckets counts approvals by time to decide.
	ApprovalBuckets  []ResponseBucket  `json:"approval_buckets"`
	RecentRejections []RejectionReason `json:"recent_rejections"`
	ExpiringSoon     []ExpiringBudget  `json:"expiring_soon"`
	Period           string            `json:"period"`
}

// ResponseBucket is a time range ("< 1 dia", "1–3 dias"...) and its count.
type ResponseBucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// RejectionReason is a recent customer rejection with its free-text reason.
type RejectionReason struct {
	BudgetID    string    `json:"budget_id"`
	BudgetName  string    `json:"budget_name"`
	QuoteNumber *int      `json:"quote_number,omitempty"`
	Customer    string    `json:"customer"`
	Reason      string    `json:"reason"`
	At          time.Time `json:"at"`
}

// ExpiringBudget is a sent budget whose validity ends soon (follow-up candidate).
type ExpiringBudget struct {
	BudgetID    string    `json:"budget_id"`
	BudgetName  string    `json:"budget_name"`
	QuoteNumber *int      `json:"quote_number,omitempty"`
	Customer    string    `json:"customer"`
	ValidUntil  time.Time `json:"valid_until"`
	Total       int64     `json:"total"`
}
