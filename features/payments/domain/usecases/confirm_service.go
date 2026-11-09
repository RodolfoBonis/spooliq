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
	First    bool
}

// Confirm verifies an order with payment_check (using the org's own credentials) and,
// when paid, marks the payment and recalculates the budget aggregate exactly once. It
// is safe to call repeatedly (idempotent) and returns ErrConfirmRetry when the attempt
// should be retried later (provider down, transient DB error).
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

	payment, err := s.repo.GetPaymentByOrderNSU(ctx, organizationID, req.OrderNSU)
	if err != nil {
		return result, ErrConfirmRetry
	}
	if payment == nil {
		s.logger.Info(ctx, "Confirm: unknown order_nsu", map[string]interface{}{"organization_id": organizationID, "source": string(source)})
		return result, nil
	}
	result.BudgetID = payment.BudgetID.String()
	alreadyPaid := payment.Status == entities.PaymentPaid
	isRedirect := source == entities.SourceRedirect

	// A pending row flagged for manual review (by a TRUSTED source) is skipped so we
	// neither re-alert nor act on it. The redirect can never set this flag (🔴1).
	if payment.Status == entities.PaymentPending && payment.NeedsReview {
		return result, nil
	}
	// An already-paid row confirmed via the public redirect does NOTHING — no
	// payment_check, no alert — so a user cannot spoof refund alerts (N3/🔴1).
	if alreadyPaid && isRedirect {
		result.State = entities.ConfirmPaidState
		return result, nil
	}

	integration, err := s.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		return result, ErrConfirmRetry
	}
	if integration == nil || integration.APIKeyCiphertext == "" {
		if !isRedirect {
			s.flagReview(ctx, payment, entities.ReviewNoCredentials)
			s.alert(ctx, "confirm without credentials — cannot verify received payment", map[string]interface{}{
				"organization_id": organizationID, "order_nsu": req.OrderNSU, "source": string(source),
			})
		}
		return result, ErrPaymentsNotConfigured
	}
	apiKey, err := s.decryptAPIKey(integration)
	if err != nil {
		return result, ErrPaymentsNotConfigured
	}

	// Identifiers sent to payment_check: ALWAYS prefer the STORED ones; a request's
	// slug/transaction_nsu is used only when the row has none (🔴1 — request values are
	// spoofable, so they never override and never trigger a mismatch rejection).
	handle := payment.Handle
	if handle == "" {
		handle = integration.Handle
	}
	storedSlug := deref(payment.Slug)
	storedTxn := deref(payment.TransactionNSU)
	slugToCheck := firstNonEmpty(storedSlug, req.ResolvedSlug())
	txnToCheck := firstNonEmpty(storedTxn, req.TransactionNSU)

	// Bump the attempt counter (backoff) for trusted retries only; never persist
	// request hints before verification. A non-23505 DB error is retried (N5).
	if !isRedirect && !alreadyPaid {
		if err := s.repo.UpdatePaymentFields(ctx, payment.ID, map[string]interface{}{
			"last_checked_at": time.Now(),
			"check_attempts":  gorm.Expr("check_attempts + 1"),
		}); err != nil {
			if isUniqueViolation(err) {
				return result, nil
			}
			return result, ErrConfirmRetry
		}
	}

	check, err := s.client.PaymentCheck(ctx, apiKey, coreServices.InfinitePayPaymentCheckRequest{
		Handle:         handle,
		OrderNSU:       req.OrderNSU,
		TransactionNSU: txnToCheck,
		Slug:           slugToCheck,
	})
	if err != nil {
		if isUnauthorized(err) {
			if !isRedirect && !alreadyPaid {
				s.flagReview(ctx, payment, entities.ReviewUnauthorized)
			}
			if !isRedirect {
				s.alert(ctx, "payment_check unauthorized", map[string]interface{}{"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "source": string(source)})
			}
			return result, ErrInvalidCredentials
		}
		return result, ErrConfirmRetry
	}

	// Bind the gateway's response to our order (trustworthy: comes from payment_check).
	if check.OrderNSU != "" && check.OrderNSU != payment.OrderNSU {
		if !isRedirect && !alreadyPaid {
			s.flagReview(ctx, payment, entities.ReviewOrderMismatch)
			s.alert(ctx, "payment_check order_nsu mismatch", map[string]interface{}{
				"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "theirs": check.OrderNSU, "source": string(source),
			})
		}
		return result, nil
	}

	paid := check.Success && check.Paid

	// Already-paid row re-checked by webhook/job (refund detection).
	if alreadyPaid {
		if !paid {
			if s.notifications != nil {
				s.notifications.Notify(organizationID, budgetUc.PossibleRefundNotification(nil, "pagamento", payment.BudgetID.String(), "refund_review:"+payment.ID.String()))
			}
			s.alert(ctx, "paid payment no longer reports paid (possible refund — manual review)", map[string]interface{}{
				"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "source": string(source),
			})
		} else if check.TransactionNSU != "" && storedTxn != "" && check.TransactionNSU != storedTxn {
			s.alert(ctx, "paid row reports a different transaction_nsu", map[string]interface{}{
				"budget_id": result.BudgetID, "order_nsu": req.OrderNSU, "ours": storedTxn, "theirs": check.TransactionNSU,
			})
		}
		result.State = entities.ConfirmPaidState
		return result, nil
	}

	if !paid {
		return result, nil
	}

	// Value divergence guard, using the Pix discount SNAPSHOT.
	pixFloor := entities.PixPrice(payment.AmountCents, payment.PixDiscountPercent) - 1
	if check.PaidAmount < pixFloor {
		s.flagReview(ctx, payment, entities.ReviewValueDivergence)
		s.alert(ctx, "payment value divergence (paid below pix floor)", map[string]interface{}{
			"budget_id": result.BudgetID, "order_nsu": req.OrderNSU,
			"paid_amount": check.PaidAmount, "expected": payment.AmountCents, "pix_floor": pixFloor, "source": string(source),
		})
		return result, nil
	}

	budget, err := s.repo.GetBudgetChargeContext(ctx, payment.BudgetID, organizationID)
	if err != nil {
		return result, ErrConfirmRetry
	}
	budgetMissing := budget == nil

	now := time.Now()
	raw, _ := json.Marshal(check)
	fields := map[string]interface{}{
		"status":            string(entities.PaymentPaid),
		"paid_amount_cents": check.PaidAmount,
		"capture_method":    check.CaptureMethod,
		"installments":      check.Installments,
		"paid_at":           now,
		"raw_event":         json.RawMessage(raw),
		"link_creating_at":  nil,
	}
	if txn := firstNonEmpty(check.TransactionNSU, storedTxn, req.TransactionNSU); txn != "" {
		fields["transaction_nsu"] = txn
	}
	if slug := firstNonEmpty(check.ResolvedSlug(), storedSlug, req.ResolvedSlug()); slug != "" {
		fields["slug"] = slug
	}
	if r := firstNonEmpty(check.ReceiptURL, receiptURL); validGatewayURL(r) {
		fields["receipt_url"] = r
	}

	var rows int64
	var paidAfterCancel bool
	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		// 🔴2/N1: lock the budget FIRST (budget -> payments order), then re-read the
		// payment UNDER the lock so paid_after_cancel reflects any cancel that committed
		// between the initial read and here.
		if !budgetMissing {
			if err := repo.LockBudget(ctx, payment.BudgetID, organizationID); err != nil {
				return err
			}
		}
		locked, err := repo.GetPaymentByOrderNSUForUpdate(ctx, organizationID, payment.OrderNSU)
		if err != nil {
			return err
		}
		if locked == nil {
			return nil // vanished; rows stays 0
		}
		settlingExpired := locked.Status == entities.PaymentCanceled && locked.StatusReason == entities.ReasonExpired
		paidAfterCancel = budgetMissing || (locked.Status == entities.PaymentCanceled && entities.IsRefundReason(locked.StatusReason))
		if paidAfterCancel {
			fields["status_reason"] = entities.ReasonPaidAfterCancel
		} else {
			fields["status_reason"] = "" // a settled expired/pending link is a normal paid row
		}

		n, err := repo.MarkPaidGuarded(ctx, locked.ID, organizationID, fields)
		if err != nil {
			return err
		}
		rows = n
		if rows == 0 {
			return nil
		}
		if settlingExpired {
			if newer, nerr := repo.GetPendingByBudgetKind(ctx, payment.BudgetID, payment.Kind); nerr == nil && newer != nil && newer.ID != locked.ID {
				if _, cerr := repo.CancelPendingByID(ctx, newer.ID, entities.ReasonSuperseded); cerr != nil {
					return cerr
				}
			}
		}
		if !budgetMissing {
			return repo.RecalcBudgetAggregate(ctx, payment.BudgetID, organizationID)
		}
		return nil
	})
	if txErr != nil {
		if isUniqueViolation(txErr) {
			s.alert(ctx, "transaction_nsu reuse rejected while settling", map[string]interface{}{"budget_id": result.BudgetID, "order_nsu": req.OrderNSU})
			return result, nil
		}
		return result, ErrConfirmRetry
	}

	result.State = entities.ConfirmPaidState
	if rows == 0 {
		return result, nil
	}
	result.First = true

	if !budgetMissing {
		budgetUc.InvalidateDashboard(ctx, s.cache, s.logger, organizationID)
	}
	s.recordPaidSideEffects(ctx, organizationID, budget, payment, check, now, paidAfterCancel, budgetMissing)
	return result, nil
}

// flagReview marks a payment for manual review so the reconcile job skips it and we
// do not re-alert on every webhook/job retry.
func (s *ConfirmService) flagReview(ctx context.Context, payment *entities.BudgetPayment, reason string) {
	_ = s.repo.UpdatePaymentFields(ctx, payment.ID, map[string]interface{}{
		"needs_review":  true,
		"review_reason": reason,
	})
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

	if !budgetMissing {
		if principal, err := s.repo.SumPaidPrincipal(ctx, payment.BudgetID); err == nil && principal > budget.TotalCost {
			s.alert(ctx, "paid principal exceeds budget total", map[string]interface{}{
				"budget_id": budgetID, "order_nsu": payment.OrderNSU, "principal": principal, "total": budget.TotalCost,
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

// deref returns the string a *string points to, or "" when nil.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
