package usecases

import (
	"context"
	"encoding/json"
	"time"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"gorm.io/gorm"
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
// when paid, marks the payment and recalculates the budget aggregate exactly once. It
// is safe to call repeatedly (idempotent). It returns ErrConfirmRetry when the attempt
// should be retried later (provider down) so the caller does NOT mark it done.
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
		return result, ErrConfirmRetry
	}
	if integration == nil || integration.APIKeyCiphertext == "" {
		// No credentials to verify with (never connected, or purged after 7 days).
		s.alert(ctx, "confirm without credentials — cannot verify received payment", map[string]interface{}{
			"organization_id": organizationID, "order_nsu": req.OrderNSU, "source": string(source),
		})
		return result, ErrPaymentsNotConfigured
	}

	payment, err := s.repo.GetPaymentByOrderNSU(ctx, organizationID, req.OrderNSU)
	if err != nil {
		return result, ErrConfirmRetry
	}
	if payment == nil {
		s.logger.Info(ctx, "Confirm: unknown order_nsu", map[string]interface{}{"organization_id": organizationID, "source": string(source)})
		return result, nil
	}
	result.BudgetID = payment.BudgetID.String()

	apiKey, err := s.decryptAPIKey(integration)
	if err != nil {
		return result, err
	}

	reqSlug := req.ResolvedSlug()
	// Slug binding: a different slug for our order_nsu means the webhook is not about
	// our link — reject it.
	if payment.Slug != nil && *payment.Slug != "" && reqSlug != "" && reqSlug != *payment.Slug {
		s.alert(ctx, "slug mismatch on confirm", map[string]interface{}{"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "source": string(source)})
		return result, nil
	}

	// Persist the webhook's slug/transaction_nsu BEFORE calling payment_check, so the
	// row carries them even if we crash, and the unique index catches a reused
	// transaction_nsu. A unique violation here means the transaction was already
	// recorded against another order — alert and stop.
	pre := map[string]interface{}{
		"last_checked_at": time.Now(),
		"check_attempts":  gorm.Expr("check_attempts + 1"),
	}
	if reqSlug != "" && (payment.Slug == nil || *payment.Slug == "") {
		pre["slug"] = reqSlug
	}
	if req.TransactionNSU != "" && (payment.TransactionNSU == nil || *payment.TransactionNSU == "") {
		pre["transaction_nsu"] = req.TransactionNSU
	}
	if err := s.repo.UpdatePaymentFields(ctx, payment.ID, pre); err != nil {
		s.alert(ctx, "failed to persist pre-check identifiers (possible transaction reuse)", map[string]interface{}{
			"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "error": err.Error(),
		})
		return result, nil
	}

	check, err := s.client.PaymentCheck(ctx, apiKey, coreServices.InfinitePayPaymentCheckRequest{
		Handle:         integration.Handle,
		OrderNSU:       req.OrderNSU,
		TransactionNSU: req.TransactionNSU,
		Slug:           reqSlug,
	})
	if err != nil {
		switch {
		case isUnauthorized(err):
			s.alert(ctx, "payment_check unauthorized", map[string]interface{}{"budget_id": result.BudgetID, "source": string(source)})
			return result, ErrInvalidCredentials
		default:
			// Provider down or ambiguous: retry later.
			return result, ErrConfirmRetry
		}
	}

	// Bind the response to our order: if it carries an order_nsu it must be ours.
	if check.OrderNSU != "" && check.OrderNSU != payment.OrderNSU {
		s.alert(ctx, "payment_check order_nsu mismatch", map[string]interface{}{
			"budget_id": result.BudgetID, "ours": payment.OrderNSU, "theirs": check.OrderNSU, "source": string(source),
		})
		return result, nil
	}

	if !(check.Success && check.Paid) {
		// Not paid. If our row is already paid, this looks like a refund/cancel — flag
		// for manual review but NEVER auto-flip to refunded (per plan).
		if payment.Status == entities.PaymentPaid {
			s.alert(ctx, "paid payment no longer reports paid (possible refund — manual review)", map[string]interface{}{
				"budget_id": result.BudgetID, "order_nsu": payment.OrderNSU, "source": string(source),
			})
			if s.notifications != nil {
				s.notifications.Notify(organizationID, budgetUc.PaidAfterCancelNotification(nil, "pagamento", payment.BudgetID.String()))
			}
			result.State = entities.ConfirmPaidState
		}
		return result, nil
	}

	// Already paid: idempotent duplicate confirmation, no side effects.
	if payment.Status == entities.PaymentPaid {
		result.State = entities.ConfirmPaidState
		return result, nil
	}

	// Value divergence guard, using the Pix discount SNAPSHOT taken at link creation.
	pixFloor := entities.PixPrice(payment.AmountCents, payment.PixDiscountPercent) - 1
	if check.PaidAmount < pixFloor {
		s.alert(ctx, "payment value divergence (paid below pix floor)", map[string]interface{}{
			"budget_id": result.BudgetID, "order_nsu": req.OrderNSU,
			"paid_amount": check.PaidAmount, "expected": payment.AmountCents, "pix_floor": pixFloor, "source": string(source),
		})
		return result, nil
	}

	paidAfterCancel := payment.Status == entities.PaymentCanceled

	budget, err := s.repo.GetBudgetChargeContext(ctx, payment.BudgetID, organizationID)
	if err != nil {
		return result, ErrConfirmRetry
	}
	budgetMissing := budget == nil
	if budgetMissing {
		// The budget was deleted/soft-deleted but money arrived: record it as paid with
		// paid_after_cancel so it surfaces as refund-due, never dropped silently.
		paidAfterCancel = true
	}

	now := time.Now()
	raw, _ := json.Marshal(check)
	fields := map[string]interface{}{
		"status":            string(entities.PaymentPaid),
		"paid_amount_cents": check.PaidAmount,
		"capture_method":    check.CaptureMethod,
		"installments":      check.Installments,
		"paid_at":           now,
		"raw_event":         json.RawMessage(raw),
	}
	if txn := firstNonEmpty(check.TransactionNSU, req.TransactionNSU); txn != "" {
		fields["transaction_nsu"] = txn
	}
	if slug := firstNonEmpty(check.ResolvedSlug(), reqSlug); slug != "" {
		fields["slug"] = slug
	}
	if r := firstNonEmpty(check.ReceiptURL, receiptURL); validGatewayURL(r) {
		fields["receipt_url"] = r
	}
	if paidAfterCancel {
		fields["status_reason"] = entities.ReasonPaidAfterCancel
	}

	var rows int64
	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		n, err := repo.MarkPaidGuarded(ctx, payment.ID, organizationID, fields)
		if err != nil {
			return err
		}
		rows = n
		if rows == 0 {
			return nil
		}
		if !budgetMissing {
			// Race-free: a single UPDATE … = (SELECT SUM …) recomputes the aggregate.
			return repo.RecalcBudgetAggregate(ctx, payment.BudgetID, organizationID)
		}
		return nil
	})
	if txErr != nil {
		return result, ErrConfirmRetry
	}

	result.State = entities.ConfirmPaidState
	if rows == 0 {
		return result, nil // lost the race; already applied
	}
	result.First = true

	if !budgetMissing {
		budgetUc.InvalidateDashboard(ctx, s.cache, s.logger, organizationID)
	}
	s.recordPaidSideEffects(ctx, organizationID, budget, payment, check, now, paidAfterCancel, budgetMissing)
	return result, nil
}

// recordPaidSideEffects fires the activity, notification and alerts for a freshly
// confirmed payment. budget may be nil (deleted budget).
func (s *ConfirmService) recordPaidSideEffects(
	ctx context.Context,
	organizationID string,
	budget *entities.BudgetChargeContext,
	payment *entities.BudgetPayment,
	check *coreServices.InfinitePayPaymentCheckResponse,
	now time.Time,
	paidAfterCancel bool,
	budgetMissing bool,
) {
	var quote *int
	name := "orçamento"
	if budget != nil {
		quote = budget.QuoteNumber
		if budget.Name != "" {
			name = budget.Name
		}
	}
	budgetID := payment.BudgetID.String()

	if s.activity != nil {
		s.activity.Record(context.Background(), budgetUc.PaymentReceivedActivity(organizationID, budgetID, name, quote, check.PaidAmount, now))
	}
	if s.notifications != nil {
		dedupe := "payment_received:" + payment.ID.String()
		s.notifications.Notify(organizationID, budgetUc.PaymentReceivedNotification(quote, name, budgetID, check.PaidAmount, dedupe))
	}

	// Principal above total: alert (possible over-collection / double charge).
	if !budgetMissing {
		if principal, err := s.repo.SumPaidPrincipal(ctx, payment.BudgetID); err == nil && principal > budget.TotalCost {
			s.alert(ctx, "paid principal exceeds budget total", map[string]interface{}{
				"budget_id": budgetID, "principal": principal, "total": budget.TotalCost,
			})
		}
	}

	if budgetMissing {
		s.alert(ctx, "payment confirmed for a missing/deleted budget (manual review)", map[string]interface{}{
			"budget_id": budgetID, "order_nsu": payment.OrderNSU, "amount": check.PaidAmount,
		})
	}
	if paidAfterCancel {
		s.alert(ctx, "payment received on a cancelled link (manual refund via InfinitePay)", map[string]interface{}{
			"budget_id": budgetID, "order_nsu": payment.OrderNSU, "amount": check.PaidAmount,
		})
		if s.notifications != nil {
			s.notifications.Notify(organizationID, budgetUc.PaidAfterCancelNotification(quote, name, budgetID))
		}
	}
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
