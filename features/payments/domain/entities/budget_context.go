package entities

import (
	"time"

	"github.com/google/uuid"
)

// BudgetChargeContext is the minimal slice of a budget the payments feature reads to
// decide and build a charge. It is loaded by the payments repository directly from
// the budgets table so the payments transaction (payment row + budget aggregate)
// stays in one place, without importing the budget use cases.
type BudgetChargeContext struct {
	ID              uuid.UUID
	OrganizationID  string
	PublicToken     *string
	Status          string
	TotalCost       int64
	AmountPaidCents int64
	RefundDueCents  int64
	DepositPercent  *float64
	ValidUntil      *time.Time
	Name            string
	QuoteNumber     *int

	// Customer contact (best-effort; may be empty). Used only to pre-fill the
	// InfinitePay checkout customer block.
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
}

// WebhookEventRecord is the reconcile job's view of a stored webhook event.
type WebhookEventRecord struct {
	ID             uuid.UUID
	OrganizationID string
	Payload        []byte
	Attempts       int
}
