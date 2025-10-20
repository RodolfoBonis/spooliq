package usecases

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	webhookEntities "github.com/RodolfoBonis/spooliq/features/webhooks/domain/entities"
	"github.com/gin-gonic/gin"
)

// AsaasWebhookUseCase handles Asaas webhook events
type AsaasWebhookUseCase struct {
	companyRepository      companyRepositories.CompanyRepository
	subscriptionRepository subscriptionRepositories.SubscriptionRepository
	logger                 logger.Logger
	webhookSecret          string
}

// NewAsaasWebhookUseCase creates a new webhook use case
func NewAsaasWebhookUseCase(
	companyRepository companyRepositories.CompanyRepository,
	subscriptionRepository subscriptionRepositories.SubscriptionRepository,
	cfg *config.AppConfig,
	logger logger.Logger,
) *AsaasWebhookUseCase {
	return &AsaasWebhookUseCase{
		companyRepository:      companyRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
		webhookSecret:          cfg.AsaasWebhookSecret,
	}
}

// HandleWebhook processes incoming Asaas webhook events
func (uc *AsaasWebhookUseCase) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Webhook received from Asaas", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		uc.logger.Error(ctx, "Failed to read webhook body", map[string]interface{}{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if uc.webhookSecret != "" {
		signature := c.GetHeader("asaas-access-token")
		if signature == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing signature"})
			return
		}

		if !uc.validateSignature(ctx, bodyBytes, signature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}
	}

	var webhookEvent webhookEntities.AsaasWebhookRequest
	if err := json.Unmarshal(bodyBytes, &webhookEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	// Log different data based on event type
	if isSubscriptionEvent(webhookEvent.Event) {
		uc.logger.Info(ctx, "Processing webhook event", map[string]interface{}{
			"event":              webhookEvent.Event,
			"subscription_id":    webhookEvent.Subscription.ID,
			"external_reference": webhookEvent.Subscription.ExternalReference,
			"customer":           webhookEvent.Subscription.Customer,
			"status":             webhookEvent.Subscription.Status,
		})
	} else {
		uc.logger.Info(ctx, "Processing webhook event", map[string]interface{}{
			"event":              webhookEvent.Event,
			"payment_id":         webhookEvent.Payment.ID,
			"external_reference": webhookEvent.Payment.ExternalReference,
			"subscription":       webhookEvent.Payment.Subscription,
			"customer":           webhookEvent.Payment.Customer,
		})
	}

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

func (uc *AsaasWebhookUseCase) validateSignature(ctx context.Context, body []byte, signature string) bool {
	webhookSecret := config.EnvAsaasWebhookSecret()
	valid := hmac.Equal([]byte(signature), []byte(webhookSecret))
	uc.logger.Info(ctx, "Signature validation", map[string]interface{}{"valid": valid})
	return valid
}

func (uc *AsaasWebhookUseCase) recordPayment(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook, status string, eventType string) error {
	// Validate ExternalReference - it should contain the organization_id
	if payment.ExternalReference == "" {
		uc.logger.Warning(ctx, "Webhook payment missing ExternalReference", map[string]interface{}{
			"payment_id":  payment.ID,
			"customer":    payment.Customer,
			"subscription": payment.Subscription,
			"event_type":  eventType,
		})
		// Skip recording if no organization_id - cannot link to company
		return nil
	}

	existing, _ := uc.subscriptionRepository.FindByAsaasPaymentID(ctx, payment.ID)

	var paymentDate *time.Time
	if payment.PaymentDate != "" {
		parsed, err := time.Parse("2006-01-02", payment.PaymentDate)
		if err == nil {
			paymentDate = &parsed
		}
	}

	dueDate, err := time.Parse("2006-01-02", payment.DueDate)
	if err != nil {
		dueDate = time.Now()
	}

	if existing != nil {
		existing.Status = status
		existing.EventType = eventType
		existing.PaymentDate = paymentDate
		existing.InvoiceURL = payment.InvoiceURL
		existing.NetValue = payment.NetValue
		existing.BillingType = payment.BillingType
		existing.Description = payment.Description
		existing.AsaasCustomerID = payment.Customer
		existing.AsaasSubscriptionID = payment.Subscription
		return uc.subscriptionRepository.UpdateByEntity(ctx, existing)
	}

	subscriptionPayment := &subscriptionEntities.SubscriptionEntity{
		OrganizationID:      payment.ExternalReference,
		AsaasPaymentID:      payment.ID,
		AsaasCustomerID:     payment.Customer,
		AsaasSubscriptionID: payment.Subscription,
		Amount:              payment.Value,
		NetValue:            payment.NetValue,
		Status:              status,
		EventType:           eventType,
		BillingType:         payment.BillingType,
		Description:         payment.Description,
		PaymentDate:         paymentDate,
		DueDate:             dueDate,
		InvoiceURL:          payment.InvoiceURL,
	}

	return uc.subscriptionRepository.Create(ctx, subscriptionPayment)
}

// isSubscriptionEvent checks if the event is related to subscriptions
func isSubscriptionEvent(event string) bool {
	subscriptionEvents := []string{
		"SUBSCRIPTION_CREATED",
		"SUBSCRIPTION_UPDATED", 
		"SUBSCRIPTION_INACTIVATED",
		"SUBSCRIPTION_DELETED",
		"SUBSCRIPTION_SPLIT_DISABLED",
		"SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK",
		"SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK_FINISHED",
	}
	
	for _, subscriptionEvent := range subscriptionEvents {
		if event == subscriptionEvent {
			return true
		}
	}
	
	return false
}

// processEvent routes webhook events to appropriate handlers
func (uc *AsaasWebhookUseCase) processEvent(ctx context.Context, event webhookEntities.AsaasWebhookRequest) error {
	switch event.Event {
	// Payment Creation and Authorization (2)
	case "PAYMENT_CREATED":
		return uc.handlePaymentCreated(ctx, event.Payment)
	case "PAYMENT_AUTHORIZED":
		return uc.handlePaymentAuthorized(ctx, event.Payment)

	// Risk Analysis (3)
	case "PAYMENT_AWAITING_RISK_ANALYSIS":
		return uc.handlePaymentAwaitingRiskAnalysis(ctx, event.Payment)
	case "PAYMENT_APPROVED_BY_RISK_ANALYSIS":
		return uc.handlePaymentApprovedByRisk(ctx, event.Payment)
	case "PAYMENT_REPROVED_BY_RISK_ANALYSIS":
		return uc.handlePaymentReprovedByRisk(ctx, event.Payment)

	// Confirmation and Receipt (2)
	case "PAYMENT_RECEIVED":
		return uc.handlePaymentReceived(ctx, event.Payment)
	case "PAYMENT_CONFIRMED":
		return uc.handlePaymentConfirmed(ctx, event.Payment)
	case "PAYMENT_ANTICIPATED":
		return uc.handlePaymentAnticipated(ctx, event.Payment)

	// Overdue (1)
	case "PAYMENT_OVERDUE":
		return uc.handlePaymentOverdue(ctx, event.Payment)

	// Updates (1)
	case "PAYMENT_UPDATED":
		return uc.handlePaymentUpdated(ctx, event.Payment)

	// Deletion and Restoration (2)
	case "PAYMENT_DELETED":
		return uc.handlePaymentDeleted(ctx, event.Payment)
	case "PAYMENT_RESTORED":
		return uc.handlePaymentRestored(ctx, event.Payment)

	// Refunds (4)
	case "PAYMENT_REFUNDED":
		return uc.handlePaymentRefunded(ctx, event.Payment)
	case "PAYMENT_PARTIALLY_REFUNDED":
		return uc.handlePaymentPartiallyRefunded(ctx, event.Payment)
	case "PAYMENT_REFUND_IN_PROGRESS":
		return uc.handlePaymentRefundInProgress(ctx, event.Payment)
	case "PAYMENT_REFUND_DENIED":
		return uc.handlePaymentRefundDenied(ctx, event.Payment)

	// Chargebacks (3)
	case "PAYMENT_CHARGEBACK_REQUESTED":
		return uc.handlePaymentChargebackRequested(ctx, event.Payment)
	case "PAYMENT_CHARGEBACK_DISPUTE":
		return uc.handlePaymentChargebackDispute(ctx, event.Payment)
	case "PAYMENT_AWAITING_CHARGEBACK_REVERSAL":
		return uc.handlePaymentAwaitingChargebackReversal(ctx, event.Payment)

	// Dunning (2)
	case "PAYMENT_DUNNING_REQUESTED":
		return uc.handlePaymentDunningRequested(ctx, event.Payment)
	case "PAYMENT_DUNNING_RECEIVED":
		return uc.handlePaymentDunningReceived(ctx, event.Payment)

	// Views - Analytics (2)
	case "PAYMENT_CHECKOUT_VIEWED":
		return uc.handlePaymentCheckoutViewed(ctx, event.Payment)
	case "PAYMENT_BANK_SLIP_VIEWED":
		return uc.handlePaymentBankSlipViewed(ctx, event.Payment)

	// Special Operations (2)
	case "PAYMENT_RECEIVED_IN_CASH_UNDONE":
		return uc.handlePaymentReceivedInCashUndone(ctx, event.Payment)
	case "PAYMENT_CREDIT_CARD_CAPTURE_REFUSED":
		return uc.handlePaymentCaptureRefused(ctx, event.Payment)

	// Split Operations (3)
	case "PAYMENT_SPLIT_CANCELLED":
		return uc.handlePaymentSplitCancelled(ctx, event.Payment)
	case "PAYMENT_SPLIT_DIVERGENCE_BLOCK":
		return uc.handlePaymentSplitBlocked(ctx, event.Payment)
	case "PAYMENT_SPLIT_DIVERGENCE_BLOCK_FINISHED":
		return uc.handlePaymentSplitUnblocked(ctx, event.Payment)

	// Subscription Events (7)
	case "SUBSCRIPTION_CREATED":
		return uc.handleSubscriptionCreated(ctx, event.Subscription)
	case "SUBSCRIPTION_UPDATED":
		return uc.handleSubscriptionUpdated(ctx, event.Subscription)
	case "SUBSCRIPTION_INACTIVATED":
		return uc.handleSubscriptionInactivated(ctx, event.Subscription)
	case "SUBSCRIPTION_DELETED":
		return uc.handleSubscriptionDeleted(ctx, event.Subscription)
	case "SUBSCRIPTION_SPLIT_DISABLED":
		return uc.handleSubscriptionSplitDisabled(ctx, event.Subscription)
	case "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK":
		return uc.handleSubscriptionSplitBlocked(ctx, event.Subscription)
	case "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK_FINISHED":
		return uc.handleSubscriptionSplitUnblocked(ctx, event.Subscription)

	default:
		uc.logger.Warning(ctx, "Unhandled webhook event type", map[string]interface{}{
			"event": event.Event,
		})
		return nil
	}
}
