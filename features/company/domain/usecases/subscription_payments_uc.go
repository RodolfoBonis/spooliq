package usecases

import (
	"net/http"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionPaymentItem is a single subscription payment record exposed to the
// owning organization. It intentionally omits gateway-internal identifiers.
type SubscriptionPaymentItem struct {
	ID          string     `json:"id"`
	Amount      float64    `json:"amount"`
	Status      string     `json:"status"`
	DueDate     time.Time  `json:"due_date"`
	PaymentDate *time.Time `json:"payment_date,omitempty"`
	InvoiceURL  string     `json:"invoice_url"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ISubscriptionPaymentsUseCase exposes the owner-facing subscription payment history.
type ISubscriptionPaymentsUseCase interface {
	ListMyPayments(c *gin.Context)
}

// SubscriptionPaymentsUseCase returns the caller organization's subscription
// payments. It reuses the subscription payment repository (the same source the
// admin payment-history endpoint reads), scoped to the org from the JWT.
type SubscriptionPaymentsUseCase struct {
	subscriptionRepository subscriptionRepositories.SubscriptionRepository
	logger                 logger.Logger
}

// NewSubscriptionPaymentsUseCase creates a new SubscriptionPaymentsUseCase.
func NewSubscriptionPaymentsUseCase(
	subscriptionRepository subscriptionRepositories.SubscriptionRepository,
	logger logger.Logger,
) ISubscriptionPaymentsUseCase {
	return &SubscriptionPaymentsUseCase{
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
	}
}

// ListMyPayments godoc
// @Summary List my subscription payments
// @Description Returns the current organization's subscription payment history, paginated with the standard envelope (Owner only).
// @Tags Company
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 20, max 100)"
// @Success 200 {object} helpers.Page[usecases.SubscriptionPaymentItem] "Paginated subscription payments"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /company/subscription/payments [get]
func (uc *SubscriptionPaymentsUseCase) ListMyPayments(c *gin.Context) {
	ctx := c.Request.Context()

	orgIDStr := helpers.GetOrganizationIDString(c)
	if orgIDStr == "" {
		coreerrors.Respond(c, coreerrors.Unauthorized("organization_id_missing", "Organização não encontrada no contexto"))
		return
	}

	organizationID, err := uuid.Parse(orgIDStr)
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_organization_id", "ID de organização inválido"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: 20})

	payments, err := uc.subscriptionRepository.FindAll(ctx, organizationID, q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch subscription payments", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgIDStr,
		})
		coreerrors.Respond(c, err)
		return
	}

	total, err := uc.subscriptionRepository.CountByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to count subscription payments", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgIDStr,
		})
		coreerrors.Respond(c, err)
		return
	}

	items := make([]SubscriptionPaymentItem, len(payments))
	for i, p := range payments {
		items[i] = SubscriptionPaymentItem{
			ID:          p.ID.String(),
			Amount:      p.Amount,
			Status:      p.Status,
			DueDate:     p.DueDate,
			PaymentDate: p.PaymentDate,
			InvoiceURL:  p.InvoiceURL,
			CreatedAt:   p.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, helpers.NewPage(items, total, q))
}
