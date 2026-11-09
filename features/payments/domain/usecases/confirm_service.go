package usecases

import (
	"context"
	"encoding/json"
	"time"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
)

// ConfirmService verifies a payment with payment_check and applies the result
// idempotently. It is used by the webhook, the public redirect and the reconcile job.
type ConfirmService struct {
	*Service
}

// NewConfirmService builds the confirm service.
func NewConfirmService(s *Service) *ConfirmService {
	return &ConfirmService{Service: s}
}

// ConfirmResult reports the outcome of a confirmation attempt.
type ConfirmResult struct {
	State    entities.ConfirmState
	BudgetID string
	// First is true when THIS call made the first (state-changing) confirmation.
	First bool
}

// Confirm verifies an order with payment_check (using the org's own credentials) and,
// when paid, marks the payment and recalculates the budget aggregate exactly once.
// It is safe to call repeatedly (idempotent): a second confirmation is a no-op.
func (s *ConfirmService) Confirm(
	ctx context.Context,
	organizationID string,
	req entities.WebhookPayload,
	receiptURL string,
	source entities.ConfirmSource,
) (ConfirmResult, error) {
	result := ConfirmResult{State: entities.ConfirmProcessingState}
	if err := s.available(); err != nil {
		return result, err
	}

	integration, err := s.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		return result, err
	}
	if integration == nil || integration.APIKeyCiphertext == "" {
		// No credentials to verify with (never connected, or purged after 7 days).
		return result, ErrPaymentsNotConfigured
	}

	payment, err := s.repo.GetPaymentByOrderNSU(ctx, organizationID, req.OrderNSU)
	if err != nil {
		return result, err
	}
	if payment == nil {
		// Unknown order for this org: nothing to confirm (ignore gracefully).
		s.logger.Info(ctx, "Confirm: unknown order_nsu", map[string]interface{}{"organization_id": organizationID, "source": string(source)})
		return result, nil
	}
	result.BudgetID = payment.BudgetID.String()

	// Already settled: idempotent no-op.
	if payment.Status == entities.PaymentPaid {
		result.State = entities.ConfirmPaidState
		return result, nil
	}

	apiKey, err := s.decryptAPIKey(integration)
	if err != nil {
		return result, err
	}

	check, err := s.client.PaymentCheck(ctx, apiKey, coreServices.InfinitePayPaymentCheckRequest{
		Handle:         integration.Handle,
		OrderNSU:       req.OrderNSU,
		TransactionNSU: req.TransactionNSU,
		Slug:           req.Slug,
	})
	// Record the attempt regardless of outcome.
	_ = s.repo.UpdatePaymentFields(ctx, payment.ID, map[string]interface{}{
		"last_checked_at": time.Now(),
		"check_attempts":  payment.CheckAttempts + 1,
	})
	if err != nil {
		switch {
		case isUnauthorized(err):
			s.alert(ctx, "payment_check unauthorized", map[string]interface{}{"budget_id": result.BudgetID, "source": string(source)})
			return result, ErrInvalidCredentials
		case isUnavailable(err):
			// Leave pending; the job/retry will pick it up.
			return result, nil
		default:
			return result, ErrProviderRejected
		}
	}

	if !paymentIsPaid(check) {
		// Not paid yet (or a non-payment event). Stay pending.
		return result, nil
	}

	// Value divergence guard: a Pix payer pays at least PixPrice-1; anything below is
	// a divergence we refuse to settle on.
	pixFloor := entities.PixPrice(payment.AmountCents, integration.PixDiscountPercent) - 1
	if check.PaidAmount < pixFloor {
		s.alert(ctx, "payment value divergence (paid below pix floor)", map[string]interface{}{
			"budget_id":   result.BudgetID,
			"order_nsu":   req.OrderNSU,
			"paid_amount": check.PaidAmount,
			"expected":    payment.AmountCents,
			"pix_floor":   pixFloor,
			"source":      string(source),
		})
		return result, nil
	}

	paidAfterCancel := payment.Status == entities.PaymentCanceled

	budget, err := s.repo.GetBudgetChargeContext(ctx, payment.BudgetID, organizationID)
	if err != nil {
		return result, err
	}
	if budget == nil {
		return result, nil
	}

	raw, _ := json.Marshal(check)
	now := time.Now()
	paidFields := map[string]interface{}{
		"status":            string(entities.PaymentPaid),
		"paid_amount_cents": check.PaidAmount,
		"capture_method":    check.CaptureMethod,
		"installments":      check.Installments,
		"paid_at":           now,
		"raw_event":         json.RawMessage(raw),
	}
	if check.TransactionNSU != "" {
		paidFields["transaction_nsu"] = check.TransactionNSU
	}
	if check.Slug != "" {
		paidFields["slug"] = check.Slug
	}
	if r := firstNonEmpty(receiptURL, check.ReceiptURL); validGatewayURL(r) {
		paidFields["receipt_url"] = r
	}
	if paidAfterCancel {
		paidFields["status_reason"] = entities.ReasonPaidAfterCancel
	}

	var rows int64
	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		n, err := repo.MarkPaidGuarded(ctx, payment.ID, paidFields)
		if err != nil {
			return err
		}
		rows = n
		if rows == 0 {
			return nil // already confirmed concurrently
		}
		paidPrincipal, err := repo.SumPaidPrincipal(ctx, payment.BudgetID)
		if err != nil {
			return err
		}
		status := entities.Status(budget.TotalCost, paidPrincipal)
		var fullyPaidAt *time.Time
		if status == entities.StatusPaid {
			fullyPaidAt = &now
		}
		return repo.UpdateBudgetAggregate(ctx, payment.BudgetID, paidPrincipal, string(status), fullyPaidAt)
	})
	if txErr != nil {
		return result, txErr
	}

	result.State = entities.ConfirmPaidState
	if rows == 0 {
		// Lost the race: another confirmation already applied this payment.
		return result, nil
	}
	result.First = true

	// First-confirmation side effects (best-effort, never fail the caller).
	budgetUc.InvalidateDashboard(ctx, s.cache, s.logger, organizationID)
	s.recordPaidSideEffects(ctx, organizationID, budget, payment, check, now, paidAfterCancel)
	return result, nil
}

// recordPaidSideEffects fires the activity, notification and (for paid-after-cancel)
// the alert for a freshly confirmed payment.
func (s *ConfirmService) recordPaidSideEffects(
	ctx context.Context,
	organizationID string,
	budget *entities.BudgetChargeContext,
	payment *entities.BudgetPayment,
	check *coreServices.InfinitePayPaymentCheckResponse,
	now time.Time,
	paidAfterCancel bool,
) {
	if s.activity != nil {
		s.activity.Record(context.Background(), budgetUc.PaymentReceivedActivity(organizationID, budget.ID.String(), budget.Name, budget.QuoteNumber, check.PaidAmount, now))
	}
	if s.notifications != nil {
		dedupe := "payment_received:" + payment.ID.String()
		s.notifications.Notify(organizationID, budgetUc.PaymentReceivedNotification(budget.QuoteNumber, budget.Name, budget.ID.String(), check.PaidAmount, dedupe))
	}
	if paidAfterCancel {
		s.alert(ctx, "payment received on a cancelled link (manual refund via InfinitePay)", map[string]interface{}{
			"budget_id": budget.ID.String(),
			"order_nsu": payment.OrderNSU,
			"amount":    check.PaidAmount,
		})
		if s.notifications != nil {
			s.notifications.Notify(organizationID, budgetUc.PaidAfterCancelNotification(budget.QuoteNumber, budget.Name, budget.ID.String()))
		}
	}
}

// paymentIsPaid interprets a payment_check response as "paid". It is defensive: a
// positive paid_amount counts as paid even if the boolean flags are absent.
func paymentIsPaid(check *coreServices.InfinitePayPaymentCheckResponse) bool {
	if check == nil {
		return false
	}
	return check.Paid || check.PaidAmount > 0
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
