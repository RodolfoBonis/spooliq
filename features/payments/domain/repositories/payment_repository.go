package repositories

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/google/uuid"
)

// PaymentRepository is the data port for the payments feature. It owns the
// payment_integrations, budget_payments and payment_webhook_events tables and reads
// the small slice of the budgets table it needs (via BudgetChargeContext) so a
// charge/confirmation can update the payment row AND the budget aggregate inside one
// transaction.
type PaymentRepository interface {
	// WithTransaction runs fn against a transaction-bound repository.
	WithTransaction(ctx context.Context, fn func(repo PaymentRepository) error) error
	// WithReconcileLock runs fn while holding a session-level advisory lock so only one
	// replica sweeps at a time. The bool reports whether the lock was acquired (fn runs
	// only then).
	WithReconcileLock(ctx context.Context, fn func() error) (bool, error)
	// LockBudget takes a row lock (SELECT ... FOR UPDATE) on the budget, establishing a
	// consistent budget->payments lock order so concurrent confirmations serialize.
	LockBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) error

	// --- Integration ---
	GetIntegration(ctx context.Context, organizationID, provider string) (*entities.PaymentIntegration, error)
	CreateIntegration(ctx context.Context, integration *entities.PaymentIntegration) error
	UpdateIntegrationFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	CountPendingLinks(ctx context.Context, organizationID string) (int64, error)
	ListDisconnectedBefore(ctx context.Context, cutoff time.Time) ([]*entities.PaymentIntegration, error)
	DeleteIntegration(ctx context.Context, id uuid.UUID) error

	// --- Budget payments ---
	GetPaymentsByBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]*entities.BudgetPayment, error)
	// GetPendingByBudgetKind returns the pending payment for (budget, kind), or (nil, nil).
	GetPendingByBudgetKind(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error)
	// GetPendingByBudgetKindForUpdate is GetPendingByBudgetKind with a row lock
	// (SELECT ... FOR UPDATE), used inside the charge transaction to serialize two
	// concurrent create-link requests for the same (budget, kind).
	GetPendingByBudgetKindForUpdate(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error)
	GetPaymentByOrderNSU(ctx context.Context, organizationID, orderNSU string) (*entities.BudgetPayment, error)
	// GetPaymentByOrderNSUForUpdate locks the row for the confirm transaction.
	GetPaymentByOrderNSUForUpdate(ctx context.Context, organizationID, orderNSU string) (*entities.BudgetPayment, error)
	CreatePayment(ctx context.Context, payment *entities.BudgetPayment) error
	UpdatePaymentFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	// MarkPaidGuarded performs the guarded UPDATE ... WHERE id = ? AND organization_id = ?
	// AND status IN ('pending','canceled'); rows==1 means this call made the first
	// confirmation.
	MarkPaidGuarded(ctx context.Context, id uuid.UUID, organizationID string, fields map[string]interface{}) (int64, error)
	// CancelPendingByID cancels a single payment only if it is still pending, returning
	// rows affected (0 means it was no longer pending — never overwrite paid).
	CancelPendingByID(ctx context.Context, id uuid.UUID, reason string) (int64, error)
	// CancelPendingForBudget cancels every PENDING payment of a budget (guarded).
	CancelPendingForBudget(ctx context.Context, budgetID uuid.UUID, organizationID, reason string) error
	// CancelPendingForOrg cancels every PENDING payment of an organization (guarded).
	CancelPendingForOrg(ctx context.Context, organizationID, reason string) error
	// ListExpiredPendingLinks lists pending payments whose link_expires_at < now.
	ListExpiredPendingLinks(ctx context.Context, now time.Time, limit int) ([]*entities.BudgetPayment, error)
	// ListPendingForRecheck lists pending payments still within their validity + 48h
	// grace that are due for another payment_check (basic time-based backoff).
	ListPendingForRecheck(ctx context.Context, now time.Time, limit int) ([]*entities.BudgetPayment, error)

	// --- Budget aggregate (budgets table) ---
	GetBudgetChargeContext(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetChargeContext, error)
	GetBudgetChargeContextByToken(ctx context.Context, token string) (*entities.BudgetChargeContext, error)
	// SumPaidPrincipal sums amount_cents over the budget's PAID payments, EXCLUDING
	// paid_after_cancel (refund-due money is not "paid" toward the total).
	SumPaidPrincipal(ctx context.Context, budgetID uuid.UUID) (int64, error)
	// RecalcBudgetAggregate recomputes amount_paid_cents, amount_refund_due_cents,
	// payment_status and paid_at from the payments rows in ONE race-free UPDATE.
	RecalcBudgetAggregate(ctx context.Context, budgetID uuid.UUID, organizationID string) error
	// HasBlockingPayments reports whether the budget has any paid or pending payment
	// (used to block deletion).
	HasBlockingPayments(ctx context.Context, budgetID uuid.UUID, organizationID string) (bool, error)

	// --- Webhook events ---
	CreateWebhookEvent(ctx context.Context, organizationID string, payload []byte, receivedAt time.Time) (uuid.UUID, error)
	MarkWebhookProcessed(ctx context.Context, id uuid.UUID) error
	MarkWebhookAttempt(ctx context.Context, id uuid.UUID, lastError string) error
	ListUnprocessedWebhooks(ctx context.Context, since time.Time, maxAttempts, limit int) ([]*entities.WebhookEventRecord, error)
	// ListAgedOutWebhooks lists unprocessed events older than before (past the retry
	// window) so the job can alert that they will never be retried.
	ListAgedOutWebhooks(ctx context.Context, before time.Time, limit int) ([]*entities.WebhookEventRecord, error)
}
