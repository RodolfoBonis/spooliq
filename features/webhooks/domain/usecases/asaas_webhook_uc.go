package usecases

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	webhookEntities "github.com/RodolfoBonis/spooliq/features/webhooks/domain/entities"
	"github.com/gin-gonic/gin"
)

// AsaasWebhookUseCase handles Asaas webhook events
type AsaasWebhookUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
	webhookSecret     string
}

// NewAsaasWebhookUseCase creates a new webhook use case
func NewAsaasWebhookUseCase(
	companyRepository companyRepositories.CompanyRepository,
	cfg *config.AppConfig,
	logger logger.Logger,
) *AsaasWebhookUseCase {
	return &AsaasWebhookUseCase{
		companyRepository: companyRepository,
		logger:            logger,
		webhookSecret:     cfg.AsaasWebhookSecret,
	}
}

// HandleWebhook processes incoming Asaas webhook events
// @Summary Handle Asaas webhook
// @Description Receives and processes payment events from Asaas
// @Tags webhooks
// @Accept json
// @Produce json
// @Param event body webhookEntities.AsaasWebhookRequest true "Webhook event"
// @Success 200 {object} map[string]string "Event processed"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid signature"
// @Failure 500 {object} map[string]string "Processing error"
// @Router /v1/webhooks/asaas [post]
func (uc *AsaasWebhookUseCase) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Webhook received from Asaas", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	// 1. Read raw body for signature validation
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		uc.logger.Error(ctx, "Failed to read webhook body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 2. Validate webhook signature
	signature := c.GetHeader("Asaas-Signature")
	if signature == "" {
		uc.logger.Error(ctx, "Missing Asaas-Signature header", nil)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing signature"})
		return
	}

	if !uc.validateSignature(ctx, bodyBytes, signature) {
		uc.logger.Error(ctx, "Invalid webhook signature", map[string]interface{}{
			"signature": signature,
		})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	// 3. Parse webhook event
	var webhookEvent webhookEntities.AsaasWebhookRequest
	if err := json.Unmarshal(bodyBytes, &webhookEvent); err != nil {
		uc.logger.Error(ctx, "Failed to parse webhook event", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	uc.logger.Info(ctx, "Processing webhook event", map[string]interface{}{
		"event":      webhookEvent.Event,
		"payment_id": webhookEvent.Payment.ID,
	})

	// 4. Process event based on type
	if err := uc.processEvent(ctx, webhookEvent); err != nil {
		uc.logger.Error(ctx, "Failed to process webhook event", map[string]interface{}{
			"error": err.Error(),
			"event": webhookEvent.Event,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Event processed successfully",
		"event":   webhookEvent.Event,
	})
}

// validateSignature validates the Asaas webhook signature
func (uc *AsaasWebhookUseCase) validateSignature(ctx context.Context, body []byte, signature string) bool {
	// IMPORTANT: In production, Asaas provides a webhook secret separate from API key
	// Use environment variable ASAAS_WEBHOOK_SECRET
	// This is a simplified implementation for demonstration

	mac := hmac.New(sha256.New, []byte(uc.webhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	valid := hmac.Equal([]byte(signature), []byte(expectedSignature))

	uc.logger.Info(ctx, "Signature validation", map[string]interface{}{
		"valid": valid,
	})

	return valid
}

// processEvent processes different webhook event types
func (uc *AsaasWebhookUseCase) processEvent(ctx context.Context, event webhookEntities.AsaasWebhookRequest) error {
	switch event.Event {
	case "PAYMENT_RECEIVED":
		return uc.handlePaymentReceived(ctx, event.Payment)
	case "PAYMENT_CONFIRMED":
		return uc.handlePaymentConfirmed(ctx, event.Payment)
	case "PAYMENT_OVERDUE":
		return uc.handlePaymentOverdue(ctx, event.Payment)
	case "PAYMENT_REFUNDED":
		return uc.handlePaymentRefunded(ctx, event.Payment)
	default:
		uc.logger.Info(ctx, "Unhandled webhook event type", map[string]interface{}{
			"event": event.Event,
		})
		return nil
	}
}

// handlePaymentReceived handles PAYMENT_RECEIVED event
func (uc *AsaasWebhookUseCase) handlePaymentReceived(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_RECEIVED", map[string]interface{}{
		"payment_id":         payment.ID,
		"external_reference": payment.ExternalReference,
	})

	// Get organization ID from external reference (should be organization_id)
	orgID := payment.ExternalReference
	if orgID == "" {
		uc.logger.Error(ctx, "Missing external_reference (organization_id)", nil)
		return nil // Don't fail, just log
	}

	// Fetch company
	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		return err
	}

	// Update company status to active if payment confirmed
	if company.SubscriptionStatus == "trial" || company.SubscriptionStatus == "suspended" {
		now := time.Now()
		company.SubscriptionStatus = "active"
		company.SubscriptionStartedAt = &now
		company.LastPaymentCheck = &now

		if err := uc.companyRepository.Update(ctx, company); err != nil {
			uc.logger.Error(ctx, "Failed to update company status", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
			})
			return err
		}

		uc.logger.Info(ctx, "Company subscription activated", map[string]interface{}{
			"organization_id": orgID,
			"status":          "active",
		})
	}

	return nil
}

// handlePaymentConfirmed handles PAYMENT_CONFIRMED event
func (uc *AsaasWebhookUseCase) handlePaymentConfirmed(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_CONFIRMED", map[string]interface{}{
		"payment_id": payment.ID,
	})
	// Similar to PAYMENT_RECEIVED
	return uc.handlePaymentReceived(ctx, payment)
}

// handlePaymentOverdue handles PAYMENT_OVERDUE event
func (uc *AsaasWebhookUseCase) handlePaymentOverdue(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_OVERDUE", map[string]interface{}{
		"payment_id":         payment.ID,
		"external_reference": payment.ExternalReference,
	})

	orgID := payment.ExternalReference
	if orgID == "" {
		return nil
	}

	// Fetch company
	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgID)
	if err != nil {
		return err
	}

	// Update company status to suspended
	company.SubscriptionStatus = "suspended"
	now := time.Now()
	company.LastPaymentCheck = &now

	if err := uc.companyRepository.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to suspend company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		return err
	}

	uc.logger.Info(ctx, "Company suspended due to overdue payment", map[string]interface{}{
		"organization_id": orgID,
	})

	return nil
}

// handlePaymentRefunded handles PAYMENT_REFUNDED event
func (uc *AsaasWebhookUseCase) handlePaymentRefunded(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_REFUNDED", map[string]interface{}{
		"payment_id": payment.ID,
	})
	// Log refund, potentially notify admin
	return nil
}
