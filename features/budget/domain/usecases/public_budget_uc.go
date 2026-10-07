package usecases

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/gin-gonic/gin"
)

// Public rate-limit buckets (per IP, per minute).
const (
	publicRateWindow    = time.Minute
	publicGetRateLimit  = 60
	publicPostRateLimit = 10
	maxUserAgentLength  = 255
)

// IPublicBudgetUseCase exposes the customer-facing (no-auth) budget endpoints.
type IPublicBudgetUseCase interface {
	GetByToken(c *gin.Context)
	GetPDF(c *gin.Context)
	Approve(c *gin.Context)
	Reject(c *gin.Context)
}

// PublicBudgetUseCase implements the public budget endpoints. It never trusts the
// caller (no auth): it looks budgets up by share token only, returns a sanitized
// view, and rate-limits per IP (fail-open when Redis is unavailable).
type PublicBudgetUseCase struct {
	budgetRepository   budgetRepo.BudgetRepository
	brandingRepository companyRepo.BrandingRepository
	pdfService         *services.PDFService
	redisService       *services.RedisService
	activityService    activityUc.IActivityService
	logger             logger.Logger
}

// NewPublicBudgetUseCase creates a new PublicBudgetUseCase.
func NewPublicBudgetUseCase(
	budgetRepository budgetRepo.BudgetRepository,
	brandingRepository companyRepo.BrandingRepository,
	pdfService *services.PDFService,
	redisService *services.RedisService,
	activityService activityUc.IActivityService,
	logger logger.Logger,
) IPublicBudgetUseCase {
	return &PublicBudgetUseCase{
		budgetRepository:   budgetRepository,
		brandingRepository: brandingRepository,
		pdfService:         pdfService,
		redisService:       redisService,
		activityService:    activityService,
		logger:             logger,
	}
}

// allowRequest applies a per-IP sliding-ish fixed-window rate limit using Redis
// INCR + EXPIRE. It FAILS OPEN: when Redis is nil or any command errors, the
// request is allowed, so a cache outage never blocks the public endpoints.
func (uc *PublicBudgetUseCase) allowRequest(ctx context.Context, bucket, ip string, limit int) bool {
	if uc.redisService == nil {
		return true
	}
	client := uc.redisService.GetClient()
	if client == nil {
		return true
	}
	key := "public_budget_rl:" + bucket + ":" + ip
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return true // fail-open
	}
	if count == 1 {
		// Best-effort expiry; ignore the error (fail-open).
		_ = client.Expire(ctx, key, publicRateWindow).Err()
	}
	return count <= int64(limit)
}

// GetByToken returns the sanitized public view for a share token.
// @Summary Public budget view
// @Description Public, sanitized budget view for a customer share link. No authentication.
// @Tags public-budgets
// @Produce json
// @Param token path string true "Public share token"
// @Success 200 {object} entities.PublicBudgetView
// @Failure 404 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /public/budgets/{token} [get]
func (uc *PublicBudgetUseCase) GetByToken(c *gin.Context) {
	ctx := c.Request.Context()
	if !uc.allowRequest(ctx, "get", c.ClientIP(), publicGetRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
		return
	}

	view, err := uc.buildPublicView(ctx, budget)
	if err != nil {
		uc.logger.Error(ctx, "Failed to build public budget view", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.JSON(http.StatusOK, view)
}

// GetPDF streams the same PDF the organization downloads, for a share token.
// @Summary Public budget PDF
// @Description Streams the budget PDF for a customer share link. No authentication.
// @Tags public-budgets
// @Produce application/pdf
// @Param token path string true "Public share token"
// @Success 200 {file} binary "application/pdf"
// @Failure 404 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /public/budgets/{token}/pdf [get]
func (uc *PublicBudgetUseCase) GetPDF(c *gin.Context) {
	ctx := c.Request.Context()
	if !uc.allowRequest(ctx, "get", c.ClientIP(), publicGetRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
		return
	}

	organizationID := budget.OrganizationID

	customer, err := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get customer info for public PDF", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	items, err := uc.budgetRepository.FindItemsByBudgetID(ctx, budget.ID)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	// Mirror generate_pdf_uc exactly: the PDF distributes markup over budget.TotalCost.
	itemsResponse, totalHours, totalMins := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.TotalCost, organizationID)

	company, err := uc.budgetRepository.GetCompanyByOrganizationID(ctx, organizationID)
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr("company_not_configured", "Informações da empresa não encontradas."))
		return
	}

	branding, err := uc.brandingRepository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		branding = nil // PDFService falls back to the default template
	}

	pdfBytes, err := uc.pdfService.GenerateBudgetPDF(ctx, services.BudgetPDFData{
		Budget:                budget,
		Customer:              customer,
		Items:                 itemsResponse,
		Company:               company,
		Branding:              branding,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate public PDF", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// publicResponseRequest is the shared body for approve/reject.
type publicResponseRequest struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Approve records a customer approval for a share token.
// @Summary Public budget approve
// @Description Customer approves a shared budget. No authentication.
// @Tags public-budgets
// @Accept json
// @Produce json
// @Param token path string true "Public share token"
// @Param request body object true "Approval payload: name"
// @Success 200 {object} entities.PublicBudgetView
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 410 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /public/budgets/{token}/approve [post]
func (uc *PublicBudgetUseCase) Approve(c *gin.Context) {
	uc.respond(c, entities.StatusApproved)
}

// Reject records a customer rejection for a share token.
// @Summary Public budget reject
// @Description Customer rejects a shared budget. No authentication.
// @Tags public-budgets
// @Accept json
// @Produce json
// @Param token path string true "Public share token"
// @Param request body object true "Rejection payload: name, reason"
// @Success 200 {object} entities.PublicBudgetView
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 410 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /public/budgets/{token}/reject [post]
func (uc *PublicBudgetUseCase) Reject(c *gin.Context) {
	uc.respond(c, entities.StatusRejected)
}

// respond is the shared approve/reject handler. newStatus is approved or rejected.
func (uc *PublicBudgetUseCase) respond(c *gin.Context, newStatus entities.BudgetStatus) {
	ctx := c.Request.Context()
	if !uc.allowRequest(ctx, "post", c.ClientIP(), publicPostRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	var request publicResponseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Requisição inválida"))
		return
	}

	name := strings.TrimSpace(request.Name)
	if len(name) < 2 || len(name) > 120 {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "Informe seu nome (entre 2 e 120 caracteres)"))
		return
	}

	var reason *string
	if newStatus == entities.StatusRejected {
		trimmed := strings.TrimSpace(request.Reason)
		if len(trimmed) > 1000 {
			coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "O motivo deve ter no máximo 1000 caracteres"))
			return
		}
		if trimmed != "" {
			reason = &trimmed
		}
	}

	budget, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
		return
	}

	now := time.Now()
	if !budget.CanRespond(now) {
		coreErrors.Respond(c, mapNotRespondableError(budget, now))
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	if len(userAgent) > maxUserAgentLength {
		userAgent = userAgent[:maxUserAgentLength]
	}

	rows, err := uc.budgetRepository.RespondToPublicBudget(ctx, budget.ID, newStatus, name, ip, userAgent, reason, now)
	if err != nil {
		uc.logger.Error(ctx, "Failed to record customer response", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if rows == 0 {
		// Lost the race (or expired between the read and the write): re-read and map.
		fresh, ferr := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
		if ferr != nil {
			coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
			return
		}
		coreErrors.Respond(c, mapNotRespondableError(fresh, time.Now()))
		return
	}

	// Record the activity (actor is the customer, suffixed to distinguish it).
	quoteNumber := 0
	if budget.QuoteNumber != nil {
		quoteNumber = *budget.QuoteNumber
	}
	action := activityEntities.ActionApproved
	if newStatus == entities.StatusRejected {
		action = activityEntities.ActionRejected
	}
	uc.activityService.Record(context.Background(), activityEntities.ActivityEntity{
		OrganizationID: budget.OrganizationID,
		UserID:         name + " (cliente)",
		Action:         action,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budget.ID.String(),
		EntityName:     budget.Name,
		Metadata: map[string]any{
			"quote_number":  quoteNumber,
			"customer_name": name,
		},
		CreatedAt: now,
	})

	updated, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	view, err := uc.buildPublicView(ctx, updated)
	if err != nil {
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	c.JSON(http.StatusOK, view)
}

// mapNotRespondableError maps a non-answerable budget to the precise error:
//   - approved/rejected => 409 budget_already_responded
//   - expired           => 410 budget_expired
//   - anything else      => 409 budget_not_available
func mapNotRespondableError(budget *entities.BudgetEntity, now time.Time) *coreErrors.APIError {
	switch {
	case budget.Status == entities.StatusApproved || budget.Status == entities.StatusRejected:
		return coreErrors.Conflict(CodeBudgetAlreadyResponded, "Este orçamento já foi respondido")
	case budget.EffectiveStatus(now) == entities.StatusExpired:
		return coreErrors.Gone(CodeBudgetExpired, "Este orçamento expirou")
	default:
		return coreErrors.Conflict(CodeBudgetNotAvailable, "Este orçamento não está disponível para resposta")
	}
}

// buildPublicView assembles the sanitized public view for a budget. It performs NO
// writes. The per-item sale values mirror the PDF (markup distributed over
// budget.TotalCost, as in generate_pdf_uc).
func (uc *PublicBudgetUseCase) buildPublicView(ctx context.Context, budget *entities.BudgetEntity) (*entities.PublicBudgetView, error) {
	now := time.Now()
	organizationID := budget.OrganizationID

	customer, err := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
	if err != nil {
		return nil, err
	}

	company, err := uc.budgetRepository.GetCompanyByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	items, err := uc.budgetRepository.GetItems(ctx, budget.ID)
	if err != nil {
		return nil, err
	}
	itemResponses, _, _ := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.TotalCost, organizationID)

	publicItems := make([]entities.PublicBudgetItem, 0, len(itemResponses))
	for _, it := range itemResponses {
		publicItems = append(publicItems, entities.PublicBudgetItem{
			ProductName:        it.ProductName,
			ProductDescription: it.ProductDescription,
			ProductQuantity:    it.ProductQuantity,
			ProductDimensions:  it.ProductDimensions,
			UnitPrice:          it.SaleUnitPrice,
			TotalPrice:         it.SaleTotal,
		})
	}

	view := &entities.PublicBudgetView{
		QuoteNumber:          budget.QuoteNumber,
		Name:                 budget.Name,
		Description:          budget.Description,
		Status:               string(budget.EffectiveStatus(now)),
		ValidUntil:           budget.ValidUntil,
		IsExpired:            budget.IsExpired(now),
		CanRespond:           budget.CanRespond(now),
		CreatedAt:            budget.CreatedAt,
		CustomerResponseAt:   budget.CustomerResponseAt,
		CustomerResponseName: budget.CustomerResponseName,
		RejectionReason:      budget.RejectionReason,
		Customer:             entities.PublicBudgetCustomer{Name: customer.Name},
		Company: entities.PublicBudgetCompany{
			Name:      company.Name,
			TradeName: company.TradeName,
			LogoURL:   company.LogoURL,
			Email:     company.Email,
			Phone:     company.Phone,
			WhatsApp:  company.WhatsApp,
			Instagram: company.Instagram,
			Website:   company.Website,
			City:      company.City,
			State:     company.State,
		},
		Items:          publicItems,
		BasePrice:      budget.BasePrice(),
		DiscountAmount: budget.DiscountAmount,
		ShippingCost:   budget.ShippingCost,
		TaxAmount:      budget.TaxAmount,
		TaxRateApplied: budget.TaxRateApplied,
		Total:          budget.TotalCost,
		DeliveryDays:   budget.DeliveryDays,
		PaymentTerms:   budget.PaymentTerms,
		Notes:          budget.Notes,
	}
	return view, nil
}
