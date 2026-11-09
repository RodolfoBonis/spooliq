package usecases

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxWebhookBody caps the webhook payload at 64 KB.
const maxWebhookBody = 64 * 1024

// confirmTimeout bounds the background confirmation triggered by a webhook.
const confirmTimeout = 20 * time.Second

// WebhookUseCase implements the InfinitePay webhook endpoint. The webhook is
// unsigned, so the per-integration token in the URL is the only credential and every
// state change is re-verified via payment_check.
type WebhookUseCase struct {
	*Service
	confirm *ConfirmService
}

// NewWebhookUseCase builds the webhook use case.
func NewWebhookUseCase(s *Service, confirm *ConfirmService) *WebhookUseCase {
	return &WebhookUseCase{Service: s, confirm: confirm}
}

// Handle receives an InfinitePay webhook, validates the token in constant time,
// durably stores the event and triggers an asynchronous confirmation. It always
// responds quickly: 200 on accept, 401 on a bad token, 413 on an oversized body.
// @Summary InfinitePay webhook
// @Tags webhooks
// @Accept json
// @Produce json
// @Param orgId path string true "Organization ID"
// @Param token path string true "Webhook token"
// @Success 200 "Accepted"
// @Failure 401 {object} errors.HTTPError
// @Failure 413 {object} errors.HTTPError
// @Router /webhooks/infinitepay/{orgId}/{token} [post]
func (uc *WebhookUseCase) Handle(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := c.Param("orgId")
	token := c.Param("token")

	if err := uc.available(); err != nil {
		coreErrors.Respond(c, coreErrors.ServiceUnavailable("payments_not_configured", "Pagamentos não estão configurados."))
		return
	}

	// Enforce the body size limit (read one extra byte to detect overflow).
	limited := io.LimitReader(c.Request.Body, maxWebhookBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Corpo inválido"))
		return
	}
	if len(body) > maxWebhookBody {
		coreErrors.Respond(c, &coreErrors.APIError{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: "Payload muito grande"})
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if integration == nil || integration.WebhookTokenCiphertext == "" {
		coreErrors.Respond(c, coreErrors.Unauthorized("invalid_webhook_token", "Token inválido"))
		return
	}
	expected, err := uc.decryptWebhookToken(integration)
	if err != nil {
		coreErrors.Respond(c, coreErrors.ServiceUnavailable("payments_not_configured", "Pagamentos não estão configurados."))
		return
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(token)) != 1 {
		coreErrors.Respond(c, coreErrors.Unauthorized("invalid_webhook_token", "Token inválido"))
		return
	}

	var payload entities.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		// Still 200 so the provider does not hammer retries on a payload we can log and
		// inspect later, but record it as an event for visibility.
		_, _ = uc.repo.CreateWebhookEvent(ctx, organizationID, body, time.Now())
		c.Status(http.StatusOK)
		return
	}

	eventID, err := uc.repo.CreateWebhookEvent(ctx, organizationID, body, time.Now())
	if err != nil {
		uc.logger.Error(ctx, "Failed to store webhook event", map[string]interface{}{"organization_id": organizationID, "error": err.Error()})
	}

	// Ack immediately; confirm asynchronously with its own bounded context.
	c.Status(http.StatusOK)

	go uc.processAsync(organizationID, payload, eventID)
}

// processAsync confirms the payment in the background and records the outcome on the
// stored event so the reconcile job can retry failures.
func (uc *WebhookUseCase) processAsync(organizationID string, payload entities.WebhookPayload, eventID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), confirmTimeout)
	defer cancel()

	_, err := uc.confirm.Confirm(ctx, organizationID, payload, "", entities.SourceWebhook)
	if eventID == uuid.Nil {
		return
	}
	if err != nil {
		_ = uc.repo.MarkWebhookAttempt(ctx, eventID, err.Error())
		return
	}
	_ = uc.repo.MarkWebhookProcessed(ctx, eventID)
}
