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
// authenticated and the public pay endpoints.
type ChargeService struct {
	*Service
}

// NewChargeService builds the charge service.
func NewChargeService(s *Service) *ChargeService {
	return &ChargeService{Service: s}
}

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
// due charge.
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
	if !payableStatus(budget.Status) {
		return nil, ErrChargeNotAvailable
	}

	next := entities.NextChargeFor(budget.TotalCost, budget.AmountPaidCents, budget.DepositPercent)
	if !next.Chargeable {
		switch next.Reason {
		case entities.ReasonAlreadyPaid:
			return nil, ErrAlreadyPaid
		case entities.ReasonBelowMinimum:
			return nil, ErrBelowMinimum
		default:
			return nil, ErrChargeNotAvailable
		}
	}
	if requestedKind != nil && *requestedKind != next.Kind {
		return nil, ErrChargeKindMismatch
	}

	now := time.Now()
	validUntil := s.linkValidUntil(budget, now)
	amount := next.Amount
	kind := next.Kind
	pix := integration.PixDiscountPercent

	var payment *entities.BudgetPayment
	var reuseURL string
	var inProgress bool

	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		// N1: lock the budget row FIRST (budget -> payments order), then the pending row.
		if err := repo.LockBudget(ctx, budget.ID, budget.OrganizationID); err != nil {
			return err
		}
		pending, err := repo.GetPendingByBudgetKindForUpdate(ctx, budget.ID, kind)
		if err != nil {
			return err
		}
		if pending != nil {
			expired := pending.LinkExpiresAt != nil && pending.LinkExpiresAt.Before(now)
			equivalent := pending.AmountCents == amount && pending.PixDiscountPercent == pix && !expired
			if equivalent {
				if pending.CheckoutURL != "" {
					payment = pending
					reuseURL = pending.CheckoutURL
					return nil
				}
				// No URL yet: honor a fresh creation lease (N4) to avoid a double
				// CreateLink; otherwise claim the lease and (re)try the remote call.
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
			// Amount/settings changed or the link expired: supersede it (guarded).
			rows, cerr := repo.CancelPendingByID(ctx, pending.ID, entities.ReasonSuperseded)
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
		if pending, perr := s.repo.GetPendingByBudgetKind(ctx, budget.ID, kind); perr == nil && pending != nil && pending.CheckoutURL != "" {
			return &entities.CreateLinkResponse{
				CheckoutURL: pending.CheckoutURL,
				Kind:        pending.Kind,
				Amount:      pending.AmountCents,
				ExpiresAt:   pending.LinkExpiresAt,
			}, nil
		}
		s.logger.Error(ctx, "Failed to persist payment link", map[string]interface{}{"budget_id": budget.ID, "error": txErr.Error()})
		return nil, txErr
	}

	if inProgress {
		return nil, ErrLinkInProgress
	}
	if reuseURL != "" {
		return &entities.CreateLinkResponse{CheckoutURL: reuseURL, Kind: kind, Amount: amount, ExpiresAt: payment.LinkExpiresAt}, nil
	}

	resp, err := s.createRemoteLink(ctx, budget, integration, payment, amount, validUntil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// createRemoteLink calls InfinitePay to create the link and persists the returned
// checkout_url/slug on the payment row, releasing the creation lease.
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
		Items:    []coreServices.InfinitePayItem{{Name: itemName, Price: amount, Quantity: 1}},
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
		Events: []string{"payment"},
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
		return nil, s.mapClientError(ctx, "create_link", budget.ID, err)
	}
	if !validGatewayURL(resp.CheckoutURL) {
		s.alert(ctx, "checkout_url rejected (non-infinitepay host)", map[string]interface{}{
			"budget_id": budget.ID.String(),
			"order_nsu": payment.OrderNSU,
		})
		return nil, ErrProviderRejected
	}

	fields := map[string]interface{}{"checkout_url": resp.CheckoutURL, "link_creating_at": nil}
	if resp.Slug != "" {
		fields["slug"] = resp.Slug
	}
	if err := s.repo.UpdatePaymentFields(ctx, payment.ID, fields); err != nil {
		s.logger.Error(ctx, "Failed to persist checkout_url", map[string]interface{}{"payment_id": payment.ID, "error": err.Error()})
	}

	return &entities.CreateLinkResponse{CheckoutURL: resp.CheckoutURL, Kind: payment.Kind, Amount: amount, ExpiresAt: &validUntil}, nil
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
