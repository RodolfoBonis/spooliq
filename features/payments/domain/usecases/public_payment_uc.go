package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Public rate limit. Pay shares the budget write bucket ("post", 10/min, together
// with approve/reject). Confirm has its OWN bucket (30/min) so a customer who
// approves and is then redirected back (which polls confirm a few times) is not
// throttled by the shared write limit. These MUST match the budget package's
// publicRateKeyPrefix / window.
const (
	publicRateKeyPrefix     = "public_budget_rl"
	publicRateBucketPost    = "post"
	publicRateBucketConfirm = "confirm"
	publicRateWindow        = time.Minute
	publicPostRateLimit     = 10
	publicConfirmRateLimit  = 30
)

// PublicConfirmResponse is the body of the public confirm endpoint: the FULL public
// budget view (same body as GET /public/budgets/:token, payment block included) plus
// a top-level `confirmation`. The embedded view's fields are promoted to the top level
// by encoding/json. The web strips `confirmation` and setQueryData's the rest.
type PublicConfirmResponse struct {
	*budgetEntities.PublicBudgetView
	Confirmation entities.ConfirmState `json:"confirmation"`
}

// PublicPaymentUseCase implements the customer-facing (no-auth) pay/confirm endpoints.
type PublicPaymentUseCase struct {
	*Service
	charge      *ChargeService
	confirm     *ConfirmService
	viewBuilder budgetUc.PublicViewBuilder
	redis       *coreServices.RedisService
}

// NewPublicPaymentUseCase builds the public payment use case.
func NewPublicPaymentUseCase(s *Service, charge *ChargeService, confirm *ConfirmService, viewBuilder budgetUc.PublicViewBuilder, redis *coreServices.RedisService) *PublicPaymentUseCase {
	return &PublicPaymentUseCase{Service: s, charge: charge, confirm: confirm, viewBuilder: viewBuilder, redis: redis}
}

func (uc *PublicPaymentUseCase) allow(c *gin.Context, bucket string, limit int) bool {
	var client *redis.Client
	if uc.redis != nil {
		client = uc.redis.GetClient()
	}
	return helpers.AllowRequest(c.Request.Context(), client, publicRateKeyPrefix, bucket, helpers.PublicClientIP(c), publicRateWindow, limit)
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
	if !uc.allow(c, publicRateBucketPost, publicPostRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests("rate_limited", "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContextByToken(ctx, c.Param("token"))
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
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
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
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
// public budget view plus the confirmation state.
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
	if !uc.allow(c, publicRateBucketConfirm, publicConfirmRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests("rate_limited", "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.repo.GetBudgetChargeContextByToken(ctx, c.Param("token"))
	if err != nil {
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
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
		uc.logger.Error(ctx, "payments endpoint internal error", map[string]interface{}{"error": err.Error()})
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

// respondConfirm reloads the budget's public view and returns it with the
// confirmation state at the top level.
func (uc *PublicPaymentUseCase) respondConfirm(c *gin.Context, token string, state entities.ConfirmState) {
	ctx := c.Request.Context()
	resp := PublicConfirmResponse{Confirmation: state}
	if uc.viewBuilder != nil {
		if view, err := uc.viewBuilder.BuildPublicViewByToken(ctx, token); err != nil {
			uc.logger.Error(ctx, "Failed to rebuild public view after confirm", map[string]interface{}{"error": err.Error()})
		} else {
			resp.PublicBudgetView = view
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
