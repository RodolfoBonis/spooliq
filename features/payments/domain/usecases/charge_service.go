package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/google/uuid"
)

// errSupersedeRace signals that a pending link we tried to supersede was no longer
// pending (it got paid/cancelled concurrently). The caller recomputes the charge.
var errSupersedeRace = errors.New("supersede_race")

// linkLeaseWindow is how long a "creating link" lease is honored before another
// request may retry the remote call for the same order.
const linkLeaseWindow = 30 * time.Second

// ChargeService creates or reuses a checkout link for a budget. It is shared by the
// authenticated and the public pay endpoints. now() is injectable for tests.
type ChargeService struct {
	*Service
	now func() time.Time
}

// NewChargeService builds the charge service.
func NewChargeService(s *Service) *ChargeService {
	return &ChargeService{Service: s, now: time.Now}
}

// SetNowForTest overrides the clock (tests only).
func (s *ChargeService) SetNowForTest(f func() time.Time) { s.now = f }

// payableStatus reports whether a budget status allows a charge.
func payableStatus(status string) bool {
	switch status {
	case "approved", "printing", "completed":
		return true
	default:
		return false
	}
}

// CreateOrReuseLink resolves the next charge for a budget and returns a checkout
// link, reusing an equivalent pending link when one exists (same amount AND same Pix
// discount snapshot, not expired). requestedKind, when non-nil, must match the next
// due charge. The budget is RE-READ under the row lock so the decision is never based
// on a stale snapshot.
func (s *ChargeService) CreateOrReuseLink(
	ctx context.Context,
	budget *entities.BudgetChargeContext,
	integration *entities.PaymentIntegration,
	requestedKind *entities.ChargeKind,
) (*entities.CreateLinkResponse, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if integration == nil || !integration.Active() {
		return nil, ErrPaymentsDisabled
	}

	now := s.now()
	pix := integration.PixDiscountPercent

	var payment *entities.BudgetPayment
	var fresh *entities.BudgetChargeContext
	var amount int64
	var kind entities.ChargeKind
	var validUntil time.Time
	var reuseURL string
	var inProgress bool
	var chargeErr error

	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		// 🟡1/N1: lock the budget FIRST, then re-read it through the tx repo and compute
		// the next charge from the locked, fresh snapshot.
		if err := repo.LockBudget(ctx, budget.ID, budget.OrganizationID); err != nil {
			return err
		}
		b, err := repo.GetBudgetChargeContext(ctx, budget.ID, budget.OrganizationID)
		if err != nil {
			return err
		}
		if b == nil {
			chargeErr = ErrChargeNotAvailable
			return nil
		}
		fresh = b
		if !payableStatus(b.Status) {
			chargeErr = ErrChargeNotAvailable
			return nil
		}
		next := entities.NextChargeFor(b.TotalCost, b.AmountPaidCents, b.DepositPercent)
		if !next.Chargeable {
			switch next.Reason {
			case entities.ReasonAlreadyPaid:
				chargeErr = ErrAlreadyPaid
			case entities.ReasonBelowMinimum:
				chargeErr = ErrBelowMinimum
			default:
				chargeErr = ErrChargeNotAvailable
			}
			return nil
		}
		if requestedKind != nil && *requestedKind != next.Kind {
			chargeErr = ErrChargeKindMismatch
			return nil
		}
		amount = next.Amount
		kind = next.Kind
		validUntil = s.linkValidUntil(b, now)

		pending, err := repo.GetPendingByBudgetKindForUpdate(ctx, budget.ID, kind)
		if err != nil {
			return err
		}
		if pending != nil {
			expired := pending.LinkExpiresAt != nil && pending.LinkExpiresAt.Before(now)
			sameValue := pending.AmountCents == amount && pending.PixDiscountPercent == pix
			if sameValue && !expired {
				if pending.CheckoutURL != "" {
					payment = pending
					reuseURL = pending.CheckoutURL
					return nil
				}
				if pending.LinkCreatingAt != nil && now.Sub(*pending.LinkCreatingAt) < linkLeaseWindow {
					inProgress = true
					return nil
				}
				if err := repo.UpdatePaymentFields(ctx, pending.ID, map[string]interface{}{"link_creating_at": now}); err != nil {
					return err
				}
				pending.LinkCreatingAt = &now
				payment = pending
				return nil
			}
			// 🟡2: supersede with ReasonExpired when the ONLY reason is expiry (same
			// value), so a late payment on it settles normally; otherwise superseded.
			reason := entities.ReasonSuperseded
			if sameValue && expired {
				reason = entities.ReasonExpired
			}
			rows, cerr := repo.CancelPendingByID(ctx, pending.ID, reason)
			if cerr != nil {
				return cerr
			}
			if rows == 0 {
				return errSupersedeRace
			}
		}
		newPayment := &entities.BudgetPayment{
			ID:                 uuid.New(),
			OrganizationID:     budget.OrganizationID,
			BudgetID:           budget.ID,
			Kind:               kind,
			OrderNSU:           genOrderNSU(),
			Handle:             integration.Handle,
			AmountCents:        amount,
			PixDiscountPercent: pix,
			Status:             entities.PaymentPending,
			LinkExpiresAt:      &validUntil,
			LinkCreatingAt:     &now,
		}
		if err := repo.CreatePayment(ctx, newPayment); err != nil {
			return err
		}
		payment = newPayment
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errSupersedeRace) {
			return nil, ErrChargeNotAvailable
		}
		if pending, perr := s.repo.GetPendingByBudgetKind(ctx, budget.ID, kindOrZero(requestedKind)); perr == nil && pending != nil && pending.CheckoutURL != "" {
			return &entities.CreateLinkResponse{CheckoutURL: pending.CheckoutURL, Kind: pending.Kind, Amount: pending.AmountCents, ExpiresAt: pending.LinkExpiresAt}, nil
		}
		s.logger.Error(ctx, "Failed to persist payment link", map[string]interface{}{"budget_id": budget.ID, "error": txErr.Error()})
		return nil, txErr
	}
	if chargeErr != nil {
		return nil, chargeErr
	}
	if inProgress {
		return nil, ErrLinkInProgress
	}
	if reuseURL != "" {
		return &entities.CreateLinkResponse{CheckoutURL: reuseURL, Kind: kind, Amount: amount, ExpiresAt: payment.LinkExpiresAt}, nil
	}

	return s.createRemoteLink(ctx, fresh, integration, payment, amount, validUntil)
}

// kindOrZero dereferences an optional kind, returning "" when nil (only used for a
// best-effort fallback lookup).
func kindOrZero(k *entities.ChargeKind) entities.ChargeKind {
	if k == nil {
		return ""
	}
	return *k
}

// createRemoteLink calls InfinitePay to create the link and persists the returned
// checkout_url/slug on the payment row (guarded to pending), releasing the lease.
func (s *ChargeService) createRemoteLink(
	ctx context.Context,
	budget *entities.BudgetChargeContext,
	integration *entities.PaymentIntegration,
	payment *entities.BudgetPayment,
	amount int64,
	validUntil time.Time,
) (*entities.CreateLinkResponse, error) {
	apiKey, err := s.decryptAPIKey(integration)
	if err != nil {
		return nil, err
	}
	webhookToken, err := s.decryptWebhookToken(integration)
	if err != nil {
		return nil, err
	}

	itemName := budget.Name
	if budget.QuoteNumber != nil && *budget.QuoteNumber > 0 {
		itemName = fmt.Sprintf("Orçamento #%d", *budget.QuoteNumber)
	}
	if itemName == "" {
		itemName = "Orçamento"
	}

	req := coreServices.InfinitePayCreateLinkRequest{
		Handle:   integration.Handle,
		Items:    []coreServices.InfinitePayItem{{Description: itemName, Price: amount, Quantity: 1}},
		OrderNSU: payment.OrderNSU,
		Options: &coreServices.InfinitePayLinkOptions{
			SinglePayment: true,
			ValidUntil:    validUntil.UTC().Format(time.RFC3339),
		},
	}
	if integration.AssuredInstallments > 0 {
		req.Options.AssuredInstallments = integration.AssuredInstallments
	}
	if integration.PixDiscountPercent > 0 {
		req.Options.Discounts = &coreServices.InfinitePayDiscounts{Pix: integration.PixDiscountPercent}
	}
	if budget.PublicToken != nil && *budget.PublicToken != "" {
		req.RedirectURL = s.redirectURL(*budget.PublicToken)
	}
	req.Webhook = &coreServices.InfinitePayWebhook{
		URL:    s.webhookURL(budget.OrganizationID, webhookToken),
		Events: []string{"paid", "refunded", "canceled", "payment_failed"},
	}
	if budget.CustomerName != "" || budget.CustomerEmail != "" {
		req.Customer = &coreServices.InfinitePayCustomer{
			Name:        budget.CustomerName,
			Email:       budget.CustomerEmail,
			PhoneNumber: budget.CustomerPhone,
		}
	}

	resp, err := s.client.CreateLink(ctx, apiKey, req)
	if err != nil {
		// 🟢2: release the lease so a retry is not blocked for the full window.
		_ = s.repo.UpdatePaymentFields(ctx, payment.ID, map[string]interface{}{"link_creating_at": nil})
		return nil, s.mapClientError(ctx, "create_link", budget.ID, err)
	}
	checkoutURL := resp.ResolvedCheckoutURL()
	if !validGatewayURL(checkoutURL) {
		_ = s.repo.UpdatePaymentFields(ctx, payment.ID, map[string]interface{}{"link_creating_at": nil})
		s.alert(ctx, "checkout_url rejected (non-infinitepay host)", map[string]interface{}{
			"budget_id": budget.ID.String(),
			"order_nsu": payment.OrderNSU,
		})
		return nil, ErrProviderRejected
	}

	fields := map[string]interface{}{"checkout_url": checkoutURL, "link_creating_at": nil}
	if resp.Slug != "" {
		fields["slug"] = resp.Slug
	}
	// 🟢1: only write the URL while the row is still pending; 0 rows => it was
	// paid/canceled concurrently, so there is nothing to charge.
	rows, uerr := s.repo.UpdatePendingFields(ctx, payment.ID, fields)
	if uerr != nil {
		s.logger.Error(ctx, "Failed to persist checkout_url", map[string]interface{}{"payment_id": payment.ID, "error": uerr.Error()})
		return nil, ErrConfirmRetry
	}
	if rows == 0 {
		return nil, ErrChargeNotAvailable
	}

	return &entities.CreateLinkResponse{CheckoutURL: checkoutURL, Kind: payment.Kind, Amount: amount, ExpiresAt: &validUntil}, nil
}

// mapClientError maps an InfinitePay client error to a payment sentinel, alerting on
// provider 5xx.
func (s *ChargeService) mapClientError(ctx context.Context, op string, budgetID uuid.UUID, err error) error {
	switch {
	case isUnauthorized(err):
		return ErrInvalidCredentials
	case isUnavailable(err):
		s.alert(ctx, "InfinitePay unavailable", map[string]interface{}{"op": op, "budget_id": budgetID.String()})
		return ErrProviderUnavailable
	default:
		return ErrProviderRejected
	}
}
