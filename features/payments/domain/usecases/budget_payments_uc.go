package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BudgetPaymentsUseCase implements the authenticated budget payment endpoints
// (GET /budgets/:id/payments and POST /budgets/:id/payments/link).
type BudgetPaymentsUseCase struct {
	*Service
	charge *ChargeService
}

// NewBudgetPaymentsUseCase builds the budget payments use case.
func NewBudgetPaymentsUseCase(s *Service, charge *ChargeService) *BudgetPaymentsUseCase {
	return &BudgetPaymentsUseCase{Service: s, charge: charge}
}

// List returns the budget's payments and aggregate.
// @Summary List budget payments
// @Tags payments
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} entities.BudgetPaymentsResponse
// @Failure 404 {object} errors.HTTPError
// @Failure 503 {object} errors.HTTPError
// @Security BearerAuth
// @Router /budgets/{id}/payments [get]
func (uc *BudgetPaymentsUseCase) List(c *gin.Context) {
	ctx := c.Request.Context()
	if err := uc.available(); err != nil {
		coreErrors.Respond(c, MapChargeError(err))
		return
	}
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não identificada"))
		return
	}
	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_budget_id", "ID de orçamento inválido"))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContext(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("budget_not_found", "Orçamento não encontrado"))
		return
	}

	payments, err := uc.repo.GetPaymentsByBudget(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	// The current fee table estimates fees for payments confirmed before it existed.
	var rates *entities.FeeRates
	if integration, ierr := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay); ierr == nil && integration != nil {
		rates = integration.FeeRates
	}

	c.JSON(http.StatusOK, buildBudgetPaymentsResponse(budget, payments, rates))
}

// CreateLink creates (or reuses) the next checkout link for the budget.
// @Summary Create budget payment link
// @Tags payments
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} entities.CreateLinkResponse
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 422 {object} errors.HTTPError
// @Failure 503 {object} errors.HTTPError
// @Security BearerAuth
// @Router /budgets/{id}/payments/link [post]
func (uc *BudgetPaymentsUseCase) CreateLink(c *gin.Context) {
	ctx := c.Request.Context()
	if err := uc.available(); err != nil {
		coreErrors.Respond(c, MapChargeError(err))
		return
	}
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não identificada"))
		return
	}
	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_budget_id", "ID de orçamento inválido"))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContext(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("budget_not_found", "Orçamento não encontrado"))
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	resp, cerr := uc.charge.CreateOrReuseLink(ctx, budget, integration, nil)
	if cerr != nil {
		coreErrors.Respond(c, MapChargeError(cerr))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// buildBudgetPaymentsResponse assembles the authenticated payments response. rates is
// the org's current fee table (may be nil); it is used to estimate per-line fees for
// payments confirmed before any table existed (the snapshot wins when present).
func buildBudgetPaymentsResponse(budget *entities.BudgetChargeContext, payments []*entities.BudgetPayment, rates *entities.FeeRates) entities.BudgetPaymentsResponse {
	paid := budget.AmountPaidCents
	balance := budget.TotalCost - paid
	if balance < 0 {
		balance = 0
	}
	lines := make([]entities.PaymentLineResponse, 0, len(payments))
	var fees, netReceived int64
	for _, p := range payments {
		feeAmount, netAmount, ratePercent := resolveLineFee(p, rates)
		lines = append(lines, entities.PaymentLineResponse{
			Kind:           p.Kind,
			Amount:         p.AmountCents,
			PaidAmount:     p.PaidAmountCents,
			CaptureMethod:  p.CaptureMethod,
			Installments:   p.Installments,
			Status:         string(p.Status),
			StatusReason:   p.StatusReason,
			CheckoutURL:    p.CheckoutURL,
			ReceiptURL:     p.ReceiptURL,
			PaidAt:         p.PaidAt,
			CreatedAt:      p.CreatedAt,
			FeeAmount:      feeAmount,
			NetAmount:      netAmount,
			FeeRatePercent: ratePercent,
		})
		// Totals count only paid lines that count toward the budget (exclude
		// paid_after_cancel, the same filter RecalcBudgetAggregate uses).
		if p.Status == entities.PaymentPaid && p.StatusReason != entities.ReasonPaidAfterCancel {
			netReceived += p.PaidAmountCents
			if feeAmount != nil {
				fees += *feeAmount
				netReceived -= *feeAmount
			}
		}
	}
	return entities.BudgetPaymentsResponse{
		Status:         entities.Status(budget.TotalCost, paid),
		Total:          budget.TotalCost,
		Paid:           paid,
		Balance:        balance,
		RefundDue:      budget.RefundDueCents,
		Fees:           fees,
		NetReceived:    netReceived,
		DepositPercent: budget.DepositPercent,
		Payments:       lines,
	}
}

// resolveLineFee returns the fee amount, net amount and rate for a payment line. A
// stored snapshot wins; otherwise a paid line is estimated from the current fee table.
// Unknown fees (no snapshot and no usable rate, or a non-paid line) return all-nil.
func resolveLineFee(p *entities.BudgetPayment, rates *entities.FeeRates) (feeAmount *int64, netAmount *int64, ratePercent *float64) {
	if p.FeeCents != nil && p.FeeRatePercent != nil {
		fee := *p.FeeCents
		net := p.PaidAmountCents - fee
		rate := *p.FeeRatePercent
		return &fee, &net, &rate
	}
	if p.Status != entities.PaymentPaid {
		return nil, nil, nil
	}
	rate, fee, ok := entities.FeeFor(rates, p.CaptureMethod, p.Installments, p.PaidAmountCents)
	if !ok {
		return nil, nil, nil
	}
	net := p.PaidAmountCents - fee
	return &fee, &net, &rate
}
