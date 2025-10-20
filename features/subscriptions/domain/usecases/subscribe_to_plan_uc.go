package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscribeRequest represents the request to subscribe to a plan
type SubscribeRequest struct {
	PlanID          uuid.UUID `json:"plan_id" binding:"required"`
	PaymentMethodID *uuid.UUID `json:"payment_method_id"` // Optional if using boleto/pix
	BillingType     string    `json:"billing_type" binding:"required,oneof=CREDIT_CARD BOLETO PIX"`
}

// SubscribeResponse represents the response after subscribing
type SubscribeResponse struct {
	SubscriptionID      string    `json:"subscription_id"`       // Asaas subscription ID
	Status              string    `json:"status"`                // ACTIVE, PENDING, etc
	PlanName            string    `json:"plan_name"`
	Value               float64   `json:"value"`
	Cycle               string    `json:"cycle"`
	NextDueDate         string    `json:"next_due_date"`
	FirstPaymentID      *string   `json:"first_payment_id,omitempty"`
	FirstPaymentInvoice *string   `json:"first_payment_invoice,omitempty"`
}

// ManageSubscriptionUseCase handles subscription management operations
type ManageSubscriptionUseCase struct {
	planRepo          repositories.SubscriptionPlanRepository
	paymentMethodRepo repositories.PaymentMethodRepository
	companyRepo       companyRepo.CompanyRepository
	asaasService      services.IAsaasService
	logger            logger.Logger
}

// NewManageSubscriptionUseCase creates a new instance
func NewManageSubscriptionUseCase(
	planRepo repositories.SubscriptionPlanRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	companyRepo companyRepo.CompanyRepository,
	asaasService services.IAsaasService,
	logger logger.Logger,
) *ManageSubscriptionUseCase {
	return &ManageSubscriptionUseCase{
		planRepo:          planRepo,
		paymentMethodRepo: paymentMethodRepo,
		companyRepo:       companyRepo,
		asaasService:      asaasService,
		logger:            logger,
	}
}

// SubscribeToPlan subscribes the organization to a plan
// @Summary Subscribe to a plan
// @Description Subscribe to a subscription plan with chosen payment method
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body SubscribeRequest true "Subscription data"
// @Success 201 {object} SubscribeResponse "Subscription created"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Plan or payment method not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/subscriptions/subscribe [post]
func (uc *ManageSubscriptionUseCase) SubscribeToPlan(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	// TODO: This endpoint needs complete refactoring for new PaymentGatewayLink structure
	// Currently disabled until:
	// 1. PaymentGatewayLinkRepository is implemented
	// 2. Logic updated to use subscription_plan_id FK in companies table
	// 3. Logic updated to store subscription_plan_id and payment_method_id in subscription_payments
	uc.logger.Error(ctx, "Subscribe to plan not yet implemented for new FK structure", map[string]interface{}{
		"organization_id": orgID,
	})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Subscription creation temporarily unavailable",
		"message": "This feature is being updated to support the new payment gateway structure",
	})
	// TODO: When implementing, follow this flow:
	// 1. Query PaymentGatewayLinkRepository by organization_id to get Asaas customer ID
	// 2. If not exists, create Asaas customer + PaymentGatewayLink record
	// 3. Create subscription in Asaas
	// 4. Update company.subscription_plan_id with the plan UUID
	// 5. Create SubscriptionPayment record with subscription_plan_id and payment_method_id FKs
}

// CancelSubscription cancels the current subscription
// @Summary Cancel subscription
// @Description Cancel the current active subscription
// @Tags subscriptions
// @Produce json
// @Success 200 {object} map[string]string "Subscription cancelled"
// @Failure 400 {object} map[string]string "No active subscription"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/subscriptions/cancel [delete]
func (uc *ManageSubscriptionUseCase) CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	// TODO: Refactor for PaymentGatewayLink structure
	uc.logger.Error(ctx, "Cancel subscription not yet implemented for new FK structure", map[string]interface{}{
		"organization_id": orgID,
	})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Subscription cancellation temporarily unavailable",
		"message": "This feature is being updated to support the new payment gateway structure",
	})
}

// GetSubscriptionStatus gets the current subscription status
// @Summary Get subscription status
// @Description Get detailed status of the current subscription
// @Tags subscriptions
// @Produce json
// @Success 200 {object} SubscribeResponse "Subscription status"
// @Failure 404 {object} map[string]string "No active subscription"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/subscriptions/status [get]
func (uc *ManageSubscriptionUseCase) GetSubscriptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	// TODO: Refactor for PaymentGatewayLink structure
	// Query company.subscription_plan_id and PaymentGatewayLink to get status
	uc.logger.Error(ctx, "Get subscription status not yet implemented for new FK structure", map[string]interface{}{
		"organization_id": orgID,
	})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Subscription status check temporarily unavailable",
		"message": "This feature is being updated to support the new payment gateway structure",
	})
}

// Helper function
func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
