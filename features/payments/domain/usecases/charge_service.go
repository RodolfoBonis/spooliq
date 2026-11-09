package usecases

import (
	"context"
	"fmt"
	"time"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/google/uuid"
)

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
// link, reusing an equivalent pending link when one exists. requestedKind, when
// non-nil, must match the next due charge (public pay passes the kind the customer
// clicked; the authenticated endpoint passes nil).
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

	var payment *entities.BudgetPayment
	var reuseURL string

	txErr := s.repo.WithTransaction(ctx, func(repo domainRepos.PaymentRepository) error {
		pending, err := repo.GetPendingByBudgetKind(ctx, budget.ID, kind)
		if err != nil {
			return err
		}
		if pending != nil {
			expired := pending.LinkExpiresAt != nil && pending.LinkExpiresAt.Before(now)
			if pending.AmountCents == amount && !expired {
				// Reuse the equivalent pending link.
				payment = pending
				reuseURL = pending.CheckoutURL
				return nil
			}
			// Amount changed or the link expired: supersede it.
			if err := repo.UpdatePaymentFields(ctx, pending.ID, map[string]interface{}{
				"status":        string(entities.PaymentCanceled),
				"status_reason": entities.ReasonSuperseded,
				"canceled_at":   now,
			}); err != nil {
				return err
			}
		}
		newPayment := &entities.BudgetPayment{
			ID:             uuid.New(),
			OrganizationID: budget.OrganizationID,
			BudgetID:       budget.ID,
			Kind:           kind,
			OrderNSU:       genOrderNSU(),
			AmountCents:    amount,
			Status:         entities.PaymentPending,
			LinkExpiresAt:  &validUntil,
		}
		if err := repo.CreatePayment(ctx, newPayment); err != nil {
			return err
		}
		payment = newPayment
		return nil
	})
	if txErr != nil {
		// A concurrent request may have won the (budget, kind) pending slot; try to
		// reuse its link instead of surfacing a raw unique-violation error.
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

	// Reusing a link that already has a checkout_url: no API call needed.
	if reuseURL != "" {
		return &entities.CreateLinkResponse{
			CheckoutURL: reuseURL,
			Kind:        kind,
			Amount:      amount,
			ExpiresAt:   payment.LinkExpiresAt,
		}, nil
	}

	// Call the provider. The row already exists with order_nsu, so a failure here
	// leaves it pending for the next attempt (same order_nsu => idempotent).
	resp, err := s.createRemoteLink(ctx, budget, integration, payment, amount, validUntil)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// createRemoteLink calls InfinitePay to create the link and persists the returned
// checkout_url/slug on the payment row.
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
		req.Options.Discounts = &coreServices.InfinitePayDiscounts{Pix: int(integration.PixDiscountPercent + 0.5)}
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

	fields := map[string]interface{}{"checkout_url": resp.CheckoutURL}
	if resp.Slug != "" {
		fields["slug"] = resp.Slug
	}
	if err := s.repo.UpdatePaymentFields(ctx, payment.ID, fields); err != nil {
		s.logger.Error(ctx, "Failed to persist checkout_url", map[string]interface{}{"payment_id": payment.ID, "error": err.Error()})
		// The link exists at the gateway; still return it so the customer can pay.
	}

	return &entities.CreateLinkResponse{
		CheckoutURL: resp.CheckoutURL,
		Kind:        payment.Kind,
		Amount:      amount,
		ExpiresAt:   &validUntil,
	}, nil
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
