package usecases

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IntegrationUseCase implements the authenticated company payment integration
// endpoints (GET/PUT/DELETE /company/payments/infinitepay).
type IntegrationUseCase struct {
	*Service
}

// NewIntegrationUseCase builds the integration use case.
func NewIntegrationUseCase(s *Service) *IntegrationUseCase {
	return &IntegrationUseCase{Service: s}
}

// Get returns the organization's InfinitePay integration.
// @Summary Get payment integration
// @Description Returns the organization's InfinitePay connection and settings.
// @Tags payments
// @Produce json
// @Success 200 {object} entities.IntegrationResponse
// @Failure 401 {object} errors.HTTPError
// @Failure 503 {object} errors.HTTPError
// @Security BearerAuth
// @Router /company/payments/infinitepay [get]
func (uc *IntegrationUseCase) Get(c *gin.Context) {
	ctx := c.Request.Context()
	if err := uc.available(); err != nil {
		coreErrors.Respond(c, MapChargeError(err))
		return
	}
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não identificada"))
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	privileged := isPrivileged(c)
	resp := entities.IntegrationResponse{Provider: entities.ProviderInfinitePay}
	if integration != nil {
		resp.Connected = integration.Connected()
		resp.Enabled = integration.Enabled
		resp.Handle = integration.Handle
		resp.AssuredInstallments = integration.AssuredInstallments
		resp.PixDiscountPercent = integration.PixDiscountPercent
		resp.DefaultDepositPercent = integration.DefaultDepositPercent
		resp.VerifiedAt = integration.VerifiedAt
		if privileged {
			resp.APIKeyLast4 = integration.APIKeyLast4
			if count, cerr := uc.repo.CountPendingLinks(ctx, organizationID); cerr == nil {
				resp.PendingLinks = &count
			}
		}
	} else {
		resp.AssuredInstallments = 1
	}

	c.JSON(http.StatusOK, resp)
}

// Upsert connects or updates the organization's InfinitePay integration.
// @Summary Update payment integration
// @Description Connects or updates the InfinitePay integration. Validates the API key.
// @Tags payments
// @Accept json
// @Produce json
// @Param request body entities.UpsertIntegrationRequest true "Integration settings"
// @Success 200 {object} entities.IntegrationResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 422 {object} errors.HTTPError
// @Failure 503 {object} errors.HTTPError
// @Security BearerAuth
// @Router /company/payments/infinitepay [put]
func (uc *IntegrationUseCase) Upsert(c *gin.Context) {
	ctx := c.Request.Context()
	if err := uc.available(); err != nil {
		coreErrors.Respond(c, MapChargeError(err))
		return
	}
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não identificada"))
		return
	}

	var req entities.UpsertIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Requisição inválida"))
		return
	}
	if err := validation.Validate(&req); err != nil {
		coreErrors.Respond(c, err)
		return
	}

	existing, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	hasNewKey := req.APIKey != nil && strings.TrimSpace(*req.APIKey) != ""
	if !hasNewKey && (existing == nil || existing.APIKeyCiphertext == "") {
		coreErrors.Respond(c, coreErrors.Unprocessable("api_key_required", "Informe a API key da InfinitePay."))
		return
	}

	now := time.Now()
	fields := map[string]interface{}{
		"handle":                  req.Handle,
		"enabled":                 req.Enabled,
		"assured_installments":    req.AssuredInstallments,
		"pix_discount_percent":    req.PixDiscountPercent,
		"default_deposit_percent": req.DefaultDepositPercent,
		"disconnected_at":         nil,
		"updated_at":              now,
	}

	var apiKeyCipher, apiKeyLast4 string
	if hasNewKey {
		apiKey := strings.TrimSpace(*req.APIKey)
		if verr := uc.validateCredential(ctx, apiKey, req.Handle); verr != nil {
			coreErrors.Respond(c, MapChargeError(verr))
			return
		}
		ciphertext, encErr := uc.box.EncryptString(apiKey)
		if encErr != nil {
			coreErrors.Respond(c, coreErrors.ServiceUnavailable("payments_not_configured", "Pagamentos não estão configurados no momento."))
			return
		}
		apiKeyCipher = ciphertext
		apiKeyLast4 = last4(apiKey)
		fields["api_key_ciphertext"] = ciphertext
		fields["api_key_last4"] = apiKeyLast4
		fields["verified_at"] = now
	}

	if existing == nil {
		token, tokErr := generateWebhookToken()
		if tokErr != nil {
			coreErrors.Respond(c, coreErrors.Internal())
			return
		}
		tokenCipher, encErr := uc.box.EncryptString(token)
		if encErr != nil {
			coreErrors.Respond(c, coreErrors.ServiceUnavailable("payments_not_configured", "Pagamentos não estão configurados no momento."))
			return
		}
		integration := &entities.PaymentIntegration{
			ID:                     uuid.New(),
			OrganizationID:         organizationID,
			Provider:               entities.ProviderInfinitePay,
			Handle:                 req.Handle,
			APIKeyCiphertext:       apiKeyCipher,
			APIKeyLast4:            apiKeyLast4,
			WebhookTokenCiphertext: tokenCipher,
			Enabled:                req.Enabled,
			AssuredInstallments:    req.AssuredInstallments,
			PixDiscountPercent:     req.PixDiscountPercent,
			DefaultDepositPercent:  req.DefaultDepositPercent,
			VerifiedAt:             &now,
		}
		if createErr := uc.repo.CreateIntegration(ctx, integration); createErr != nil {
			coreErrors.Respond(c, coreErrors.Internal())
			return
		}
	} else {
		// Keep the existing webhook token stable (existing links already use it);
		// only generate one if somehow missing.
		if existing.WebhookTokenCiphertext == "" {
			if token, tokErr := generateWebhookToken(); tokErr == nil {
				if tokenCipher, encErr := uc.box.EncryptString(token); encErr == nil {
					fields["webhook_token_ciphertext"] = tokenCipher
				}
			}
		}
		if updErr := uc.repo.UpdateIntegrationFields(ctx, existing.ID, fields); updErr != nil {
			coreErrors.Respond(c, coreErrors.Internal())
			return
		}
	}

	uc.Get(c)
}

// Delete disconnects the integration: it disables it, cancels pending links and keeps
// the credentials for 7 days (so late confirmations still verify), after which the
// reconcile job purges them.
// @Summary Disconnect payment integration
// @Tags payments
// @Produce json
// @Success 204 "Disconnected"
// @Failure 503 {object} errors.HTTPError
// @Security BearerAuth
// @Router /company/payments/infinitepay [delete]
func (uc *IntegrationUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	if err := uc.available(); err != nil {
		coreErrors.Respond(c, MapChargeError(err))
		return
	}
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não identificada"))
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if integration == nil {
		c.Status(http.StatusNoContent)
		return
	}

	now := time.Now()
	txErr := uc.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		if ferr := repo.UpdateIntegrationFields(ctx, integration.ID, map[string]interface{}{
			"enabled":         false,
			"disconnected_at": now,
			"updated_at":      now,
		}); ferr != nil {
			return ferr
		}
		return repo.CancelPendingForOrg(ctx, organizationID, entities.ReasonDisconnected)
	})
	if txErr != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.Status(http.StatusNoContent)
}

// validateCredential checks an API key against InfinitePay without side effects by
// issuing a payment_check for a non-existent order: a 401/403 means the key is
// rejected; any other non-5xx outcome means the key was accepted. A 5xx surfaces as
// "provider unavailable".
func (uc *IntegrationUseCase) validateCredential(ctx context.Context, apiKey, handle string) error {
	_, err := uc.client.PaymentCheck(ctx, apiKey, coreServices.InfinitePayPaymentCheckRequest{
		Handle:   handle,
		OrderNSU: genOrderNSU(),
	})
	switch {
	case err == nil:
		return nil
	case isUnauthorized(err):
		return ErrInvalidCredentials
	case isUnavailable(err):
		return ErrProviderUnavailable
	default:
		return nil
	}
}

// last4 returns the last 4 characters of a key (or the whole key when shorter).
func last4(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[len(key)-4:]
}

// generateWebhookToken returns 32 random bytes as URL-safe base64 (no padding).
func generateWebhookToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// isPrivileged reports whether the caller is an Owner or OrgAdmin (the roles allowed
// to see the API key last4 and pending links).
func isPrivileged(c *gin.Context) bool {
	for _, r := range helpers.GetUserRoles(c) {
		if r == roles.OwnerRole || r == roles.OrgAdminRole {
			return true
		}
	}
	return false
}
