package usecases

import (
	"context"

	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
)

// PublicPaymentProvider builds the "payment" block embedded in the public budget
// view. It implements the budget package's PublicPaymentProvider interface so the
// budget view can show payment state without importing the payments feature.
type PublicPaymentProvider struct {
	*Service
}

// NewPublicPaymentProvider builds the provider.
func NewPublicPaymentProvider(s *Service) *PublicPaymentProvider {
	return &PublicPaymentProvider{Service: s}
}

// PublicPaymentBlock assembles the payment block for a budget, or returns nil when
// payments were never set up for the organization (no integration and no payments).
func (p *PublicPaymentProvider) PublicPaymentBlock(ctx context.Context, budget *budgetEntities.BudgetEntity) (interface{}, error) {
	if err := p.available(); err != nil {
		return nil, nil // feature off / not configured: omit the block
	}

	integration, err := p.repo.GetIntegration(ctx, budget.OrganizationID, entities.ProviderInfinitePay)
	if err != nil {
		return nil, err
	}

	payments, err := p.repo.GetPaymentsByBudget(ctx, budget.ID, budget.OrganizationID)
	if err != nil {
		return nil, err
	}

	// No integration and nothing paid: there is nothing to show.
	if integration == nil && len(payments) == 0 {
		return nil, nil
	}

	total := budget.TotalCost
	paid := budget.AmountPaidCents
	balance := total - paid
	if balance < 0 {
		balance = 0
	}

	block := &entities.PublicPaymentBlock{
		Enabled:        integration != nil && integration.Active(),
		Status:         entities.Status(total, paid),
		Total:          total,
		Paid:           paid,
		Balance:        balance,
		DepositPercent: budget.DepositPercent,
		Payments:       publicPaymentLines(payments),
	}

	next := entities.NextChargeFor(total, paid, budget.DepositPercent)

	// Resolve can_pay and the unavailable reason.
	switch {
	case next.Reason == entities.ReasonAlreadyPaid:
		block.CanPay = false
		block.UnavailableReason = entities.UnavailableAlreadyPaid
	case !block.Enabled:
		block.CanPay = false
		block.UnavailableReason = entities.UnavailablePaymentsOff
		block.Next = p.buildNext(integration, next)
	case budget.Status == budgetEntities.StatusSent:
		// Preview before approval.
		block.CanPay = false
		block.UnavailableReason = entities.UnavailableNotApproved
		block.Next = p.buildNext(integration, next)
	case !budget.PayableStatus():
		block.CanPay = false
		block.UnavailableReason = entities.UnavailableNotPayable
	case !next.Chargeable:
		block.CanPay = false
		if next.Reason == entities.ReasonBelowMinimum {
			block.UnavailableReason = entities.UnavailableBelowMinimum
		} else {
			block.UnavailableReason = entities.UnavailableNotPayable
		}
		block.Next = p.buildNext(integration, next)
	default:
		block.CanPay = true
		block.Next = p.buildNext(integration, next)
		if pending, perr := p.repo.GetPendingByBudgetKind(ctx, budget.ID, next.Kind); perr == nil && pending != nil && pending.CheckoutURL != "" {
			block.HasPendingLink = true
		}
	}

	return block, nil
}

// buildNext builds the "next" charge hint. Returns nil for a non-chargeable,
// already-paid state.
func (p *PublicPaymentProvider) buildNext(integration *entities.PaymentIntegration, next entities.NextCharge) *entities.PublicPaymentNext {
	if next.Reason == entities.ReasonAlreadyPaid {
		return nil
	}
	assured := 1
	var pixPct int
	if integration != nil {
		if integration.AssuredInstallments > 0 {
			assured = integration.AssuredInstallments
		}
		pixPct = integration.PixDiscountPercent
	}
	return &entities.PublicPaymentNext{
		Kind:                     next.Kind,
		Amount:                   next.Amount,
		PixAmount:                entities.PixPrice(next.Amount, pixPct),
		PixDiscountPercent:       pixPct,
		InterestFreeInstallments: assured,
		InstallmentAmount:        entities.InterestFreeInstallment(next.Amount, assured),
		MaxInstallments:          entities.MaxInstallments,
	}
}

// publicPaymentLines maps PAID payments to the sanitized public rows. Receipt URLs
// are only exposed when they point to InfinitePay over https. paid_after_cancel rows
// are EXCLUDED from the public view: they are not counted toward the balance, so the
// customer must never see a "paid" receipt that does not reduce what they owe (they
// remain visible in the authenticated/owner view as refund_due).
func publicPaymentLines(payments []*entities.BudgetPayment) []entities.PublicPaymentLine {
	lines := make([]entities.PublicPaymentLine, 0)
	for _, pay := range payments {
		if pay.Status != entities.PaymentPaid || pay.StatusReason == entities.ReasonPaidAfterCancel {
			continue
		}
		var receipt *string
		if pay.ReceiptURL != nil && validGatewayURL(*pay.ReceiptURL) {
			receipt = pay.ReceiptURL
		}
		lines = append(lines, entities.PublicPaymentLine{
			Kind:          pay.Kind,
			Amount:        pay.AmountCents,
			PaidAmount:    pay.PaidAmountCents,
			CaptureMethod: pay.CaptureMethod,
			Installments:  pay.Installments,
			PaidAt:        pay.PaidAt,
			ReceiptURL:    receipt,
		})
	}
	return lines
}
