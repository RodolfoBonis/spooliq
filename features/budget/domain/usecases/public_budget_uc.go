package usecases

import (
	"context"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Public rate-limit buckets (per IP, per minute).
const (
	publicRateWindow    = time.Minute
	publicGetRateLimit  = 60
	publicPostRateLimit = 10
	maxUserAgentLength  = 255
	// publicRateKeyPrefix groups the public budget rate-limit buckets. The payment
	// endpoints reuse the same prefix/bucket so pay/approve/reject share the 10/min
	// per-IP write limit.
	publicRateKeyPrefix = "public_budget_rl"
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
	notifications      notificationUc.INotificationService
	cache              DashboardCacheInvalidator
	logger             logger.Logger
	// paymentProvider assembles the public "payment" block for a budget view. It is
	// optional (nil disables the block) and wired by the payments feature via FX, so
	// the budget package never imports it (one-directional dependency).
	paymentProvider PublicPaymentProvider
}

// PublicPaymentProvider builds the sanitized public "payment" block for a budget.
// It is implemented by the payments feature and injected so the public budget view
// can embed payment state without the budget package importing payments.
type PublicPaymentProvider interface {
	PublicPaymentBlock(ctx context.Context, budget *entities.BudgetEntity) (interface{}, error)
}

// SetPaymentProvider wires the optional public payment provider after construction.
// It is called by the payments feature's FX wiring; a nil provider leaves the
// "payment" block absent from the public view.
func (uc *PublicBudgetUseCase) SetPaymentProvider(p PublicPaymentProvider) {
	uc.paymentProvider = p
}

// DashboardCacheInvalidator clears an organization's cached dashboard after
// writes that do not go through an authenticated route (public link, jobs).
type DashboardCacheInvalidator interface {
	InvalidatePrefix(ctx context.Context, orgID, prefix string) error
}

// DashboardCachePrefix is the cache group of every dashboard endpoint.
const DashboardCachePrefix = "dashboard"

// InvalidateDashboard clears the organization's dashboard cache, logging failures.
func InvalidateDashboard(ctx context.Context, cache DashboardCacheInvalidator, log logger.Logger, orgID string) {
	if cache == nil {
		return
	}
	if err := cache.InvalidatePrefix(ctx, orgID, DashboardCachePrefix); err != nil {
		log.Error(ctx, "Failed to invalidate dashboard cache", map[string]interface{}{
			"organization_id": orgID,
			"error":           err.Error(),
		})
	}
}

// NewPublicBudgetUseCase creates a new PublicBudgetUseCase.
func NewPublicBudgetUseCase(
	budgetRepository budgetRepo.BudgetRepository,
	brandingRepository companyRepo.BrandingRepository,
	pdfService *services.PDFService,
	redisService *services.RedisService,
	activityService activityUc.IActivityService,
	notifications notificationUc.INotificationService,
	cache *middlewares.CacheMiddleware,
	logger logger.Logger,
) IPublicBudgetUseCase {
	uc := &PublicBudgetUseCase{
		budgetRepository:   budgetRepository,
		brandingRepository: brandingRepository,
		pdfService:         pdfService,
		redisService:       redisService,
		activityService:    activityService,
		notifications:      notifications,
		logger:             logger,
	}
	if cache != nil { // avoid a non-nil interface wrapping a nil pointer
		uc.cache = cache
	}
	return uc
}

// allowRequest applies the shared per-IP rate limit. It FAILS OPEN when Redis is
// unavailable (see helpers.AllowRequest).
func (uc *PublicBudgetUseCase) allowRequest(ctx context.Context, bucket, ip string, limit int) bool {
	var client *redis.Client
	if uc.redisService != nil {
		client = uc.redisService.GetClient()
	}
	return helpers.AllowRequest(ctx, client, publicRateKeyPrefix, bucket, ip, publicRateWindow, limit)
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
	if !uc.allowRequest(ctx, "get", helpers.PublicClientIP(c), publicGetRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
		return
	}
	if budget.Status == entities.StatusDraft {
		coreErrors.Respond(c, underReviewError())
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
	if !uc.allowRequest(ctx, "get", helpers.PublicClientIP(c), publicGetRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	budget, err := uc.budgetRepository.FindByPublicToken(ctx, c.Param("token"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodePublicBudgetNotFound, "Orçamento não encontrado"))
		return
	}
	if budget.Status == entities.StatusDraft {
		coreErrors.Respond(c, underReviewError())
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

	// Customer-facing documents distribute the markup over base_price so the per-item
	// totals reconcile with the Subtotal line (see generate_pdf_uc).
	itemsResponse, totalHours, totalMins, err := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.BasePrice(), organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to build budget items response for public PDF", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

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
		PublicView:            true,
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
	if !uc.allowRequest(ctx, "post", helpers.PublicClientIP(c), publicPostRateLimit) {
		coreErrors.Respond(c, coreErrors.TooManyRequests(CodeRateLimited, "Muitas requisições. Tente novamente em instantes."))
		return
	}

	var request publicResponseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeInvalidRequest, "Requisição inválida"))
		return
	}

	name := strings.TrimSpace(request.Name)
	if n := utf8.RuneCountInString(name); n < 2 || n > 120 {
		coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "Informe seu nome (entre 2 e 120 caracteres)"))
		return
	}

	var reason *string
	if newStatus == entities.StatusRejected {
		trimmed := strings.TrimSpace(request.Reason)
		if utf8.RuneCountInString(trimmed) > 1000 {
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
	if budget.Status == entities.StatusDraft {
		coreErrors.Respond(c, underReviewError())
		return
	}

	now := time.Now()
	if !budget.CanRespond(now) {
		coreErrors.Respond(c, mapNotRespondableError(budget, now))
		return
	}

	ip := helpers.PublicClientIP(c)
	userAgent := c.Request.UserAgent()
	if utf8.RuneCountInString(userAgent) > maxUserAgentLength {
		userAgent = string([]rune(userAgent)[:maxUserAgentLength])
	}

	// The response and its status-history row (which feeds the funnel, approval
	// rate and response times) are written atomically.
	note := "Resposta do cliente pelo link"
	if reason != nil {
		note = "Motivo: " + *reason
	}
	var rows int64
	err = uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		var err error
		rows, err = repo.RespondToPublicBudget(ctx, budget.ID, newStatus, name, ip, userAgent, reason, now)
		if err != nil || rows == 0 {
			return err
		}
		return repo.AddStatusHistory(ctx, &entities.BudgetStatusHistoryEntity{
			ID:             uuid.New(),
			BudgetID:       budget.ID,
			OrganizationID: budget.OrganizationID,
			PreviousStatus: entities.StatusSent,
			NewStatus:      newStatus,
			ChangedBy:      name + " (cliente)",
			Notes:          note,
			CreatedAt:      now,
		})
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to record customer response", map[string]interface{}{"error": err.Error(), "budget_id": budget.ID})
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
	InvalidateDashboard(ctx, uc.cache, uc.logger, budget.OrganizationID)

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
	if uc.notifications != nil {
		uc.notifications.Notify(budget.OrganizationID, CustomerResponseNotification(budget, newStatus, name, reason))
	}

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
// writes. The per-item sale values mirror the PDF: the markup is distributed over
// base_price, so the item total_price values sum EXACTLY to base_price (the
// "Subtotal") and Subtotal - discount + shipping + tax == total.
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
	itemResponses, _, _, err := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.BasePrice(), organizationID)
	if err != nil {
		return nil, err
	}

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

	// Embed the optional payment block (nil provider => absent). A failure here must
	// not break the budget view, so it is logged and omitted.
	if uc.paymentProvider != nil {
		if block, perr := uc.paymentProvider.PublicPaymentBlock(ctx, budget); perr != nil {
			uc.logger.Error(ctx, "Failed to build public payment block", map[string]interface{}{"error": perr.Error(), "budget_id": budget.ID})
		} else if block != nil {
			view.Payment = block
		}
	}

	return view, nil
}

// underReviewError hides a budget that went back to draft for a revision: the
// customer must not see (or answer) a half-edited version on the share link.
func underReviewError() error {
	return coreErrors.Conflict(CodeBudgetUnderReview,
		"Este orçamento está sendo revisado. Você receberá a nova versão em breve.")
}
