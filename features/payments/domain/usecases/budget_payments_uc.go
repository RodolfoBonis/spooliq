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
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("budget_not_found", "Orçamento não encontrado"))
		return
	}

	payments, err := uc.repo.GetPaymentsByBudget(ctx, budgetID, organizationID)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.JSON(http.StatusOK, buildBudgetPaymentsResponse(budget, payments))
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
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("budget_not_found", "Orçamento não encontrado"))
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, organizationID, entities.ProviderInfinitePay)
	if err != nil {
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

// buildBudgetPaymentsResponse assembles the authenticated payments response.
func buildBudgetPaymentsResponse(budget *entities.BudgetChargeContext, payments []*entities.BudgetPayment) entities.BudgetPaymentsResponse {
	paid := budget.AmountPaidCents
	balance := budget.TotalCost - paid
	if balance < 0 {
		balance = 0
	}
	lines := make([]entities.PaymentLineResponse, 0, len(payments))
	for _, p := range payments {
		lines = append(lines, entities.PaymentLineResponse{
			Kind:          p.Kind,
			Amount:        p.AmountCents,
			PaidAmount:    p.PaidAmountCents,
			CaptureMethod: p.CaptureMethod,
			Installments:  p.Installments,
			Status:        string(p.Status),
			CheckoutURL:   p.CheckoutURL,
			ReceiptURL:    p.ReceiptURL,
			PaidAt:        p.PaidAt,
			CreatedAt:     p.CreatedAt,
		})
	}
	return entities.BudgetPaymentsResponse{
		Status:         entities.Status(budget.TotalCost, paid),
		Total:          budget.TotalCost,
		Paid:           paid,
		Balance:        balance,
		DepositPercent: budget.DepositPercent,
		Payments:       lines,
	}
}
