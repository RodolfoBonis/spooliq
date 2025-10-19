package usecases

import (
	"context"
	"time"

	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	webhookEntities "github.com/RodolfoBonis/spooliq/features/webhooks/domain/entities"
)

// ============================================================================
// PAYMENT EVENT HANDLERS (28 handlers)
// ============================================================================

// Creation and Authorization (2)
func (uc *AsaasWebhookUseCase) handlePaymentCreated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_CREATED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusPending, "PAYMENT_CREATED")
}

func (uc *AsaasWebhookUseCase) handlePaymentAuthorized(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_AUTHORIZED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusAuthorized, "PAYMENT_AUTHORIZED")
}

// Risk Analysis (3)
func (uc *AsaasWebhookUseCase) handlePaymentAwaitingRiskAnalysis(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_AWAITING_RISK_ANALYSIS", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusAwaitingRiskAnalysis, "PAYMENT_AWAITING_RISK_ANALYSIS")
}

func (uc *AsaasWebhookUseCase) handlePaymentApprovedByRisk(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_APPROVED_BY_RISK_ANALYSIS", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusApprovedByRisk, "PAYMENT_APPROVED_BY_RISK_ANALYSIS")
}

func (uc *AsaasWebhookUseCase) handlePaymentReprovedByRisk(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_REPROVED_BY_RISK_ANALYSIS", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusReprovedByRisk, "PAYMENT_REPROVED_BY_RISK_ANALYSIS"); err != nil {
		return err
	}
	// Suspend company if payment reproved
	return uc.suspendCompanyIfNeeded(ctx, payment.ExternalReference)
}

// Confirmation and Receipt (5)
func (uc *AsaasWebhookUseCase) handlePaymentReceived(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_RECEIVED", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusReceived, "PAYMENT_RECEIVED"); err != nil {
		return err
	}
	return uc.activateCompanyIfNeeded(ctx, payment.ExternalReference)
}

func (uc *AsaasWebhookUseCase) handlePaymentConfirmed(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_CONFIRMED", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusConfirmed, "PAYMENT_CONFIRMED"); err != nil {
		return err
	}
	return uc.activateCompanyIfNeeded(ctx, payment.ExternalReference)
}

func (uc *AsaasWebhookUseCase) handlePaymentAnticipated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_ANTICIPATED", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusAnticipated, "PAYMENT_ANTICIPATED"); err != nil {
		return err
	}
	return uc.activateCompanyIfNeeded(ctx, payment.ExternalReference)
}

// Overdue (1)
func (uc *AsaasWebhookUseCase) handlePaymentOverdue(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_OVERDUE", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusOverdue, "PAYMENT_OVERDUE"); err != nil {
		return err
	}
	// Optionally suspend company on overdue
	return uc.suspendCompanyIfNeeded(ctx, payment.ExternalReference)
}

// Updates (1)
func (uc *AsaasWebhookUseCase) handlePaymentUpdated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_UPDATED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusUpdated, "PAYMENT_UPDATED")
}

// Deletion and Restoration (2)
func (uc *AsaasWebhookUseCase) handlePaymentDeleted(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_DELETED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusDeleted, "PAYMENT_DELETED")
}

func (uc *AsaasWebhookUseCase) handlePaymentRestored(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_RESTORED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusRestored, "PAYMENT_RESTORED")
}

// Refunds (5)
func (uc *AsaasWebhookUseCase) handlePaymentRefunded(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_REFUNDED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusRefunded, "PAYMENT_REFUNDED")
}

func (uc *AsaasWebhookUseCase) handlePaymentPartiallyRefunded(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_PARTIALLY_REFUNDED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusPartiallyRefunded, "PAYMENT_PARTIALLY_REFUNDED")
}

func (uc *AsaasWebhookUseCase) handlePaymentRefundInProgress(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_REFUND_IN_PROGRESS", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusRefundInProgress, "PAYMENT_REFUND_IN_PROGRESS")
}

func (uc *AsaasWebhookUseCase) handlePaymentRefundDenied(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_REFUND_DENIED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusRefundDenied, "PAYMENT_REFUND_DENIED")
}

// Chargebacks (3)
func (uc *AsaasWebhookUseCase) handlePaymentChargebackRequested(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Warning(ctx, "CHARGEBACK REQUESTED - Admin action may be needed", map[string]interface{}{
		"payment_id": payment.ID,
		"org_id":     payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusChargebackRequested, "PAYMENT_CHARGEBACK_REQUESTED")
}

func (uc *AsaasWebhookUseCase) handlePaymentChargebackDispute(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_CHARGEBACK_DISPUTE", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusChargebackDispute, "PAYMENT_CHARGEBACK_DISPUTE")
}

func (uc *AsaasWebhookUseCase) handlePaymentAwaitingChargebackReversal(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_AWAITING_CHARGEBACK_REVERSAL", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusAwaitingChargebackReversal, "PAYMENT_AWAITING_CHARGEBACK_REVERSAL")
}

// Dunning (2)
func (uc *AsaasWebhookUseCase) handlePaymentDunningRequested(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_DUNNING_REQUESTED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusDunningRequested, "PAYMENT_DUNNING_REQUESTED")
}

func (uc *AsaasWebhookUseCase) handlePaymentDunningReceived(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_DUNNING_RECEIVED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusDunningReceived, "PAYMENT_DUNNING_RECEIVED")
}

// Views - Analytics only (2)
func (uc *AsaasWebhookUseCase) handlePaymentCheckoutViewed(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Analytics: PAYMENT_CHECKOUT_VIEWED", map[string]interface{}{
		"payment_id": payment.ID,
		"org_id":     payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusCheckoutViewed, "PAYMENT_CHECKOUT_VIEWED")
}

func (uc *AsaasWebhookUseCase) handlePaymentBankSlipViewed(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Analytics: PAYMENT_BANK_SLIP_VIEWED", map[string]interface{}{
		"payment_id": payment.ID,
		"org_id":     payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusBankSlipViewed, "PAYMENT_BANK_SLIP_VIEWED")
}

// Special Operations (2)
func (uc *AsaasWebhookUseCase) handlePaymentReceivedInCashUndone(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_RECEIVED_IN_CASH_UNDONE", map[string]interface{}{"payment_id": payment.ID})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.StatusReceivedInCashUndone, "PAYMENT_RECEIVED_IN_CASH_UNDONE"); err != nil {
		return err
	}
	return uc.suspendCompanyIfNeeded(ctx, payment.ExternalReference)
}

func (uc *AsaasWebhookUseCase) handlePaymentCaptureRefused(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Warning(ctx, "CAPTURE REFUSED - Notify user to update payment method", map[string]interface{}{
		"payment_id": payment.ID,
		"org_id":     payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusCaptureRefused, "PAYMENT_CREDIT_CARD_CAPTURE_REFUSED")
}

// Split Operations (3)
func (uc *AsaasWebhookUseCase) handlePaymentSplitCancelled(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_SPLIT_CANCELLED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusSplitCancelled, "PAYMENT_SPLIT_CANCELLED")
}

func (uc *AsaasWebhookUseCase) handlePaymentSplitBlocked(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Warning(ctx, "PAYMENT SPLIT BLOCKED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusSplitBlocked, "PAYMENT_SPLIT_DIVERGENCE_BLOCK")
}

func (uc *AsaasWebhookUseCase) handlePaymentSplitUnblocked(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing PAYMENT_SPLIT_DIVERGENCE_BLOCK_FINISHED", map[string]interface{}{"payment_id": payment.ID})
	return uc.recordPayment(ctx, payment, subscriptionEntities.StatusSplitUnblocked, "PAYMENT_SPLIT_DIVERGENCE_BLOCK_FINISHED")
}

// ============================================================================
// SUBSCRIPTION EVENT HANDLERS (7 handlers)
// ============================================================================

func (uc *AsaasWebhookUseCase) handleSubscriptionCreated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_CREATED", map[string]interface{}{
		"subscription_id": payment.Subscription,
		"org_id":          payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionCreated, "SUBSCRIPTION_CREATED")
}

func (uc *AsaasWebhookUseCase) handleSubscriptionUpdated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_UPDATED", map[string]interface{}{
		"subscription_id": payment.Subscription,
		"org_id":          payment.ExternalReference,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionUpdated, "SUBSCRIPTION_UPDATED")
}

func (uc *AsaasWebhookUseCase) handleSubscriptionInactivated(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_INACTIVATED", map[string]interface{}{
		"subscription_id": payment.Subscription,
		"org_id":          payment.ExternalReference,
	})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionInactivated, "SUBSCRIPTION_INACTIVATED"); err != nil {
		return err
	}
	return uc.suspendCompanyIfNeeded(ctx, payment.ExternalReference)
}

func (uc *AsaasWebhookUseCase) handleSubscriptionDeleted(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_DELETED", map[string]interface{}{
		"subscription_id": payment.Subscription,
		"org_id":          payment.ExternalReference,
	})
	if err := uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionDeleted, "SUBSCRIPTION_DELETED"); err != nil {
		return err
	}
	return uc.suspendCompanyIfNeeded(ctx, payment.ExternalReference)
}

func (uc *AsaasWebhookUseCase) handleSubscriptionSplitDisabled(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_SPLIT_DISABLED", map[string]interface{}{
		"subscription_id": payment.Subscription,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionSplitDisabled, "SUBSCRIPTION_SPLIT_DISABLED")
}

func (uc *AsaasWebhookUseCase) handleSubscriptionSplitBlocked(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Warning(ctx, "SUBSCRIPTION SPLIT BLOCKED", map[string]interface{}{
		"subscription_id": payment.Subscription,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionSplitBlocked, "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK")
}

func (uc *AsaasWebhookUseCase) handleSubscriptionSplitUnblocked(ctx context.Context, payment webhookEntities.AsaasPaymentWebhook) error {
	uc.logger.Info(ctx, "Processing SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK_FINISHED", map[string]interface{}{
		"subscription_id": payment.Subscription,
	})
	return uc.recordPayment(ctx, payment, subscriptionEntities.SubscriptionSplitUnblocked, "SUBSCRIPTION_SPLIT_DIVERGENCE_BLOCK_FINISHED")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (uc *AsaasWebhookUseCase) activateCompanyIfNeeded(ctx context.Context, orgID string) error {
	if orgID == "" {
		return nil
	}

	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgID)
	if err != nil || company == nil {
		return err
	}

	if company.SubscriptionStatus == "trial" || company.SubscriptionStatus == "suspended" {
		company.SubscriptionStatus = "active"
		now := time.Now()
		company.SubscriptionStartedAt = &now
		company.LastPaymentCheck = &now

		if err := uc.companyRepository.Update(ctx, company); err != nil {
			uc.logger.Error(ctx, "Failed to activate company", map[string]interface{}{
				"error":  err.Error(),
				"org_id": orgID,
			})
			return err
		}

		uc.logger.Info(ctx, "Company subscription activated", map[string]interface{}{
			"org_id": orgID,
		})
	}

	return nil
}

func (uc *AsaasWebhookUseCase) suspendCompanyIfNeeded(ctx context.Context, orgID string) error {
	if orgID == "" {
		return nil
	}

	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgID)
	if err != nil || company == nil {
		return err
	}

	if company.SubscriptionStatus == "active" {
		company.SubscriptionStatus = "suspended"
		now := time.Now()
		company.LastPaymentCheck = &now

		if err := uc.companyRepository.Update(ctx, company); err != nil {
			uc.logger.Error(ctx, "Failed to suspend company", map[string]interface{}{
				"error":  err.Error(),
				"org_id": orgID,
			})
			return err
		}

		uc.logger.Info(ctx, "Company subscription suspended", map[string]interface{}{
			"org_id": orgID,
		})
	}

	return nil
}
