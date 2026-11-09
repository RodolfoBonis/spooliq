package di

import (
	"context"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NewSecretBox builds the payments SecretBox from the configured encryption key. An
// empty key yields a not-configured box (payments degrade to 503). An invalid key is
// logged and ALSO degrades to not-configured rather than aborting startup, so a
// misconfiguration never takes the whole API down.
func NewSecretBox(cfg *config.AppConfig, log logger.Logger) *crypto.SecretBox {
	box, err := crypto.NewSecretBox(cfg.PaymentsEncryptionKey)
	if err != nil {
		log.Error(context.Background(), "Invalid PAYMENTS_ENCRYPTION_KEY; payment endpoints disabled", map[string]interface{}{
			"error": err.Error(),
		})
		box, _ = crypto.NewSecretBox("")
	}
	return box
}

// defaultsProvider implements budgetUc.PaymentDefaultsProvider over the payment
// integration: a new budget inherits the org's default deposit percent.
type defaultsProvider struct {
	repo domainRepos.PaymentRepository
}

// NewDefaultsProvider builds the deposit-defaults provider for the budget feature.
func NewDefaultsProvider(repo domainRepos.PaymentRepository) budgetUc.PaymentDefaultsProvider {
	return &defaultsProvider{repo: repo}
}

func (p *defaultsProvider) DefaultDepositPercent(ctx context.Context, organizationID string) (*float64, error) {
	integration, err := p.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, nil
	}
	return integration.DefaultDepositPercent, nil
}

// pendingCanceler implements budgetUc.PendingPaymentCanceler. It cancels pending
// payment links on the transaction-bound *gorm.DB handed in by the budget status
// change, so the cancellation commits atomically with the status transition.
type pendingCanceler struct{}

// NewPendingCanceler builds the pending-payment canceler for the budget feature.
func NewPendingCanceler() budgetUc.PendingPaymentCanceler {
	return &pendingCanceler{}
}

func (c *pendingCanceler) CancelPending(ctx context.Context, tx *gorm.DB, budgetID uuid.UUID, organizationID, reason string) error {
	return tx.WithContext(ctx).
		Table("budget_payments").
		Where("budget_id = ? AND organization_id = ? AND status = ?", budgetID, organizationID, string(entities.PaymentPending)).
		Updates(map[string]interface{}{
			"status":        string(entities.PaymentCanceled),
			"status_reason": reason,
			"canceled_at":   time.Now(),
			"updated_at":    time.Now(),
		}).Error
}
