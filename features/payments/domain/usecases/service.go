package usecases

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// defaultLinkValidity is the fallback link lifetime when the budget has no validity.
const defaultLinkValidity = 30 * 24 * time.Hour

// maxLinkValidity caps the link lifetime at one year.
const maxLinkValidity = 365 * 24 * time.Hour

// gatewayHostSuffix is the only host accepted in checkout/receipt URLs.
const gatewayHostSuffix = "infinitepay.io"

// Service holds the collaborators shared by every payment use case. The concrete
// use cases embed it.
type Service struct {
	repo          domainRepos.PaymentRepository
	client        services.IInfinitePayClient
	box           *crypto.SecretBox
	cfg           *config.AppConfig
	notifications notificationUc.INotificationService
	activity      activityUc.IActivityService
	cache         budgetUc.DashboardCacheInvalidator
	logger        logger.Logger
}

// NewService builds the shared payment service.
func NewService(
	repo domainRepos.PaymentRepository,
	client services.IInfinitePayClient,
	box *crypto.SecretBox,
	cfg *config.AppConfig,
	notifications notificationUc.INotificationService,
	activity activityUc.IActivityService,
	cache budgetUc.DashboardCacheInvalidator,
	log logger.Logger,
) *Service {
	return &Service{
		repo:          repo,
		client:        client,
		box:           box,
		cfg:           cfg,
		notifications: notifications,
		activity:      activity,
		cache:         cache,
		logger:        log,
	}
}

// available reports whether the payment feature can operate at all: the global kill
// switch is on AND an encryption key is configured. Both failures surface as
// ErrPaymentsNotConfigured (503).
func (s *Service) available() error {
	if s.cfg == nil || !s.cfg.InfinitePayEnabled {
		return ErrPaymentsNotConfigured
	}
	if s.box == nil || !s.box.Configured() {
		return ErrPaymentsNotConfigured
	}
	return nil
}

// decryptAPIKey returns the plaintext API key for an integration. Failures are
// reported as ErrPaymentsNotConfigured (the stored ciphertext cannot be read).
func (s *Service) decryptAPIKey(integration *entities.PaymentIntegration) (string, error) {
	key, err := s.box.DecryptString(integration.APIKeyCiphertext)
	if err != nil {
		return "", ErrPaymentsNotConfigured
	}
	return key, nil
}

// decryptWebhookToken returns the plaintext webhook token for an integration.
func (s *Service) decryptWebhookToken(integration *entities.PaymentIntegration) (string, error) {
	tok, err := s.box.DecryptString(integration.WebhookTokenCiphertext)
	if err != nil {
		return "", ErrPaymentsNotConfigured
	}
	return tok, nil
}

// genOrderNSU builds a new, unique order_nsu ("spq_" + 32 hex chars = 36 chars,
// well under the 64-char column and any realistic gateway limit).
func genOrderNSU() string {
	return "spq_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// linkValidUntil resolves a link's expiry: the budget validity when set, else
// now+30d, capped at now+1y.
func (s *Service) linkValidUntil(budget *entities.BudgetChargeContext, now time.Time) time.Time {
	candidate := now.Add(defaultLinkValidity)
	if budget.ValidUntil != nil && budget.ValidUntil.After(now) {
		candidate = *budget.ValidUntil
	}
	if max := now.Add(maxLinkValidity); candidate.After(max) {
		candidate = max
	}
	return candidate
}

// webhookURL builds the InfinitePay webhook URL carrying the per-integration token.
func (s *Service) webhookURL(organizationID, webhookToken string) string {
	base := strings.TrimRight(s.cfg.PublicAPIBaseURL, "/")
	return fmt.Sprintf("%s/v1/webhooks/infinitepay/%s/%s", base, url.PathEscape(organizationID), url.PathEscape(webhookToken))
}

// redirectURL builds the post-checkout redirect back to the public budget page.
func (s *Service) redirectURL(publicToken string) string {
	base := strings.TrimRight(s.cfg.PublicWebBaseURL, "/")
	return fmt.Sprintf("%s/orcamento/%s?pagamento=retorno", base, url.PathEscape(publicToken))
}

// validGatewayURL reports whether raw is an https URL whose host is (a subdomain of)
// infinitepay.io. checkout_url and receipt_url are only trusted when this holds.
func validGatewayURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == gatewayHostSuffix || strings.HasSuffix(host, "."+gatewayHostSuffix)
}

// alert logs an error and reports it to Sentry with structured tags, so money
// anomalies (value divergence, paid-after-cancel, provider 5xx, webhook exhaustion,
// unconfirmed refunds) are visible. The webhook token is never passed in here.
func (s *Service) alert(ctx context.Context, event string, fields map[string]interface{}) {
	if s.logger != nil {
		s.logger.Error(ctx, event, fields)
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetTag("area", "payments")
		for k, v := range fields {
			scope.SetExtra(k, v)
		}
		sentry.CaptureMessage("payments: " + event)
	})
}
