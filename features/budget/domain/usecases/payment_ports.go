package usecases

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentDefaultsProvider supplies an organization's default deposit percent for new
// budgets. It is a narrow port implemented by the payments feature and injected via
// FX, so the budget package never imports payments (one-directional dependency).
type PaymentDefaultsProvider interface {
	// DefaultDepositPercent returns the org's default deposit percent, or (nil, nil)
	// when none is configured (meaning "charge the full total").
	DefaultDepositPercent(ctx context.Context, organizationID string) (*float64, error)
}

// PendingPaymentCanceler cancels a budget's pending payment links. CancelPending runs
// INSIDE the status-change transaction (it receives the transaction-bound *gorm.DB),
// so cancelling pending links and the status transition commit or roll back together.
type PendingPaymentCanceler interface {
	CancelPending(ctx context.Context, tx *gorm.DB, budgetID uuid.UUID, organizationID, reason string) error
}

// PaymentPresenceChecker reports whether a budget has payments that block deletion
// (any paid — including paid_after_cancel — or pending row). Implemented by the
// payments feature.
type PaymentPresenceChecker interface {
	HasBlockingPayments(ctx context.Context, budgetID uuid.UUID, organizationID string) (bool, error)
}

// Pending-cancel reasons passed by the budget status change.
const (
	PendingCancelBudgetCancelled = "budget_cancelled"
	PendingCancelBudgetRevised   = "budget_revised"
)
