package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Public rate limit shares the budget endpoints' bucket (10 writes/min per IP), so
// pay/confirm/approve/reject together are limited. These MUST match the budget
// package's publicRateKeyPrefix / window.
const (
	publicRateKeyPrefix  = "public_budget_rl"
	publicRateBucketPost = "post"
	publicRateWindow     = time.Minute
	publicPostRateLimit  = 10
)

// PublicConfirmResponse is the body of the public confirm endpoint. It carries the
// refreshed payment block (merge into the cached public budget view) plus the
// confirmation state.
type PublicConfirmResponse struct {
	Confirmation entities.ConfirmState        `json:"confirmation"`
	Payment      *entities.PublicPaymentBlock `json:"payment"`
}

// PublicPaymentUseCase implements the customer-facing (no-auth) pay/confirm endpoints.
type PublicPaymentUseCase struct {
	*Service
	charge   *ChargeService
	confirm  *ConfirmService
	provider *PublicPaymentProvider
	redis    *coreServices.RedisService
}

// NewPublicPaymentUseCase builds the public payment use case.
func NewPublicPaymentUseCase(s *Service, charge *ChargeService, confirm *ConfirmService, provider *PublicPaymentProvider, redis *coreServices.RedisService) *PublicPaymentUseCase {
	return &PublicPaymentUseCase{Service: s, charge: charge, confirm: confirm, provider: provider, redis: redis}
}

func (uc *PublicPaymentUseCase) allowPost(c *gin.Context) bool {
	var client *redis.Client
	if uc.redis != nil {
		client = uc.redis.GetClient()
	}
	return helpers.AllowRequest(c.Request.Context(), client, publicRateKeyPrefix, publicRateBucketPost, helpers.PublicClientIP(c), publicRateWindow, publicPostRateLimit)
}

// Pay creates a checkout link for a public budget token.
// @Summary Public create payment
// @Tags public-payments
// @Accept json
// @Produce json
// @Param token path string true "Public share token"
// @Param request body entities.CreatePublicPaymentRequest true "Payment kind"
// @Success 200 {object} entities.CreateLinkResponse
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 422 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Failure 503 {object} errors.HTTPError
// @Router /public/budgets/{token}/payments [post]
func (uc *PublicPaymentUseCase) Pay(c *gin.Context) {
	ctx := c.Request.Context()
	if !uc.allowPost(c) {
		coreErrors.Respond(c, coreErrors.TooManyRequests("rate_limited", "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContextByToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("public_budget_not_found", "Orçamento não encontrado"))
		return
	}
	if budget.Status == string(budgetEntities.StatusDraft) {
		coreErrors.Respond(c, coreErrors.Conflict("budget_under_review", "Este orçamento está sendo revisado. Você receberá a nova versão em breve."))
		return
	}

	var req entities.CreatePublicPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Requisição inválida"))
		return
	}
	if !validKind(req.Kind) {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "Tipo de pagamento inválido"))
		return
	}

	integration, err := uc.repo.GetIntegration(ctx, budget.OrganizationID, entities.ProviderInfinitePay)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	kind := req.Kind
	resp, cerr := uc.charge.CreateOrReuseLink(ctx, budget, integration, &kind)
	if cerr != nil {
		coreErrors.Respond(c, MapChargeError(cerr))
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Confirm verifies a payment from the public redirect and returns the refreshed
// payment block plus the confirmation state.
// @Summary Public confirm payment
// @Tags public-payments
// @Accept json
// @Produce json
// @Param token path string true "Public share token"
// @Param request body entities.ConfirmPublicPaymentRequest true "Confirmation identifiers"
// @Success 200 {object} PublicConfirmResponse
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /public/budgets/{token}/payments/confirm [post]
func (uc *PublicPaymentUseCase) Confirm(c *gin.Context) {
	ctx := c.Request.Context()
	if !uc.allowPost(c) {
		coreErrors.Respond(c, coreErrors.TooManyRequests("rate_limited", "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContextByToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if budget == nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("public_budget_not_found", "Orçamento não encontrado"))
		return
	}
	if budget.Status == string(budgetEntities.StatusDraft) {
		coreErrors.Respond(c, coreErrors.Conflict("budget_under_review", "Este orçamento está sendo revisado."))
		return
	}

	var req entities.ConfirmPublicPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Requisição inválida"))
		return
	}
	if req.OrderNSU == "" {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "order_nsu é obrigatório"))
		return
	}

	// Ownership: the order must belong to this token's budget.
	payment, err := uc.repo.GetPaymentByOrderNSU(ctx, budget.OrganizationID, req.OrderNSU)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if payment == nil || payment.BudgetID != budget.ID {
		coreErrors.Respond(c, coreErrors.Conflict("payment_not_available", "Pagamento não disponível para este orçamento."))
		return
	}

	result, cerr := uc.confirm.Confirm(ctx, budget.OrganizationID, entities.WebhookPayload{
		OrderNSU:       req.OrderNSU,
		TransactionNSU: req.TransactionNSU,
		Slug:           req.Slug,
	}, req.ReceiptURL, entities.SourceRedirect)
	if cerr != nil {
		// Provider hiccups degrade to "processing" so the page can keep polling.
		uc.logger.Error(ctx, "Public confirm failed", map[string]interface{}{"budget_id": budget.ID.String(), "error": cerr.Error()})
		uc.respondConfirm(c, c.Param("token"), entities.ConfirmProcessingState)
		return
	}

	uc.respondConfirm(c, c.Param("token"), result.State)
}

// respondConfirm reloads the budget and returns the refreshed payment block.
func (uc *PublicPaymentUseCase) respondConfirm(c *gin.Context, token string, state entities.ConfirmState) {
	ctx := c.Request.Context()
	resp := PublicConfirmResponse{Confirmation: state}
	if fresh, err := uc.repo.GetBudgetChargeContextByToken(ctx, token); err == nil && fresh != nil {
		minimal := &budgetEntities.BudgetEntity{
			ID:              fresh.ID,
			OrganizationID:  fresh.OrganizationID,
			Status:          budgetEntities.BudgetStatus(fresh.Status),
			TotalCost:       fresh.TotalCost,
			AmountPaidCents: fresh.AmountPaidCents,
			DepositPercent:  fresh.DepositPercent,
		}
		if block, berr := uc.provider.PublicPaymentBlock(ctx, minimal); berr == nil {
			if b, ok := block.(*entities.PublicPaymentBlock); ok {
				resp.Payment = b
			}
		}
	}
	c.JSON(http.StatusOK, resp)
}

// validKind reports whether k is an acceptable charge kind from a client.
func validKind(k entities.ChargeKind) bool {
	switch k {
	case entities.KindDeposit, entities.KindBalance, entities.KindFull:
		return true
	default:
		return false
	}
}
