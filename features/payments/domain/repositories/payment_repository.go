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

	// --- Integration ---

	// GetIntegration returns the org's integration for a provider, or (nil, nil) when
	// none exists.
	GetIntegration(ctx context.Context, organizationID, provider string) (*entities.PaymentIntegration, error)
	// CreateIntegration inserts a new integration row.
	CreateIntegration(ctx context.Context, integration *entities.PaymentIntegration) error
	// UpdateIntegrationFields updates the given columns with a map (never a struct, so
	// zero values are not skipped).
	UpdateIntegrationFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	// CountPendingLinks counts pending payment links across the organization.
	CountPendingLinks(ctx context.Context, organizationID string) (int64, error)
	// ListDisconnectedBefore lists integrations disconnected at or before cutoff
	// (used by the reconcile job to purge stale credentials).
	ListDisconnectedBefore(ctx context.Context, cutoff time.Time) ([]*entities.PaymentIntegration, error)
	// DeleteIntegration permanently removes an integration row.
	DeleteIntegration(ctx context.Context, id uuid.UUID) error

	// --- Budget payments ---

	// GetPaymentsByBudget returns all payments of a budget, newest first.
	GetPaymentsByBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]*entities.BudgetPayment, error)
	// GetPendingByBudgetKind returns the single pending payment for (budget, kind), or
	// (nil, nil) when none.
	GetPendingByBudgetKind(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error)
	// GetPaymentByOrderNSU returns a payment by its order_nsu scoped to the org, or
	// (nil, nil) when none.
	GetPaymentByOrderNSU(ctx context.Context, organizationID, orderNSU string) (*entities.BudgetPayment, error)
	// CreatePayment inserts a payment row. A unique partial index enforces at most one
	// pending row per (budget, kind).
	CreatePayment(ctx context.Context, payment *entities.BudgetPayment) error
	// UpdatePaymentFields updates the given columns of a payment by id.
	UpdatePaymentFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	// MarkPaidGuarded performs the guarded UPDATE ... WHERE id = ? AND status IN
	// ('pending','canceled'), returning rows affected (1 => this call made the first
	// confirmation). fields carry the paid columns.
	MarkPaidGuarded(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (int64, error)
	// CancelPendingForBudget cancels every pending payment of a budget with the given
	// status_reason. Runs inside the caller's transaction when WithTransaction wraps it.
	CancelPendingForBudget(ctx context.Context, budgetID uuid.UUID, organizationID, reason string) error
	// CancelPendingForOrg cancels every pending payment of an organization.
	CancelPendingForOrg(ctx context.Context, organizationID, reason string) error
	// ListExpiredPendingLinks lists pending payments whose link_expires_at < now.
	ListExpiredPendingLinks(ctx context.Context, now time.Time, limit int) ([]*entities.BudgetPayment, error)
	// ListPendingWithIdentifiers lists pending payments created since cutoff that have a
	// slug or transaction_nsu (so the job can re-check them).
	ListPendingWithIdentifiers(ctx context.Context, since time.Time, limit int) ([]*entities.BudgetPayment, error)

	// --- Budget aggregate (budgets table) ---

	// GetBudgetChargeContext loads the budget slice needed to charge, scoped to the
	// org. Returns (nil, nil) when not found.
	GetBudgetChargeContext(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetChargeContext, error)
	// GetBudgetChargeContextByToken loads the budget slice by public share token.
	// Returns (nil, nil) when not found.
	GetBudgetChargeContextByToken(ctx context.Context, token string) (*entities.BudgetChargeContext, error)
	// SumPaidPrincipal sums amount_cents over the budget's PAID payments.
	SumPaidPrincipal(ctx context.Context, budgetID uuid.UUID) (int64, error)
	// UpdateBudgetAggregate writes amount_paid_cents, payment_status and paid_at on the
	// budget.
	UpdateBudgetAggregate(ctx context.Context, budgetID uuid.UUID, amountPaid int64, status string, paidAt *time.Time) error

	// --- Webhook events ---

	// CreateWebhookEvent stores a received payload, returning its id.
	CreateWebhookEvent(ctx context.Context, organizationID string, payload []byte, receivedAt time.Time) (uuid.UUID, error)
	// MarkWebhookProcessed stamps processed_at.
	MarkWebhookProcessed(ctx context.Context, id uuid.UUID) error
	// MarkWebhookAttempt increments attempts and records the last error.
	MarkWebhookAttempt(ctx context.Context, id uuid.UUID, lastError string) error
	// ListUnprocessedWebhooks lists unprocessed events received since cutoff with fewer
	// than maxAttempts attempts.
	ListUnprocessedWebhooks(ctx context.Context, since time.Time, maxAttempts, limit int) ([]*entities.WebhookEventRecord, error)
}
