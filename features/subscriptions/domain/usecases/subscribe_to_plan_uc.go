package usecases

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyEntities "github.com/RodolfoBonis/spooliq/features/company/domain/entities"
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

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid subscribe request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// 1. Validate plan exists and is active
	plan, err := uc.planRepo.FindByID(ctx, req.PlanID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find plan", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": req.PlanID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find plan"})
		return
	}

	if plan == nil || !plan.IsActive {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found or inactive"})
		return
	}

	// 2. Get company
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find company"})
		return
	}

	if company == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	// 3. Check if company already has an active subscription
	if company.AsaasSubscriptionID != nil && *company.AsaasSubscriptionID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Company already has an active subscription. Cancel it first or use change-plan endpoint."})
		return
	}

	// 4. Ensure company has Asaas customer ID
	asaasCustomerID := ""
	if company.AsaasCustomerID == nil || *company.AsaasCustomerID == "" {
		asaasCustomerID, err = uc.createAsaasCustomer(ctx, company, orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment customer"})
			return
		}
		company.AsaasCustomerID = &asaasCustomerID
	} else {
		asaasCustomerID = *company.AsaasCustomerID
	}

	// 5. Get payment method token (if using credit card)
	var creditCardToken string
	if req.BillingType == "CREDIT_CARD" {
		if req.PaymentMethodID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment method ID is required for credit card payments"})
			return
		}

		paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, *req.PaymentMethodID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
				"error": err.Error(),
				"id":    req.PaymentMethodID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find payment method"})
			return
		}

		if paymentMethod == nil || paymentMethod.OrganizationID != orgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
			return
		}

		creditCardToken = paymentMethod.AsaasCreditCardToken
	}

	// 6. Create subscription in Asaas
	nextDueDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02") // Next month

	subscriptionReq := services.AsaasSubscriptionRequest{
		Customer:          asaasCustomerID,
		BillingType:       req.BillingType,
		Value:             plan.Price,
		NextDueDate:       nextDueDate,
		Cycle:             plan.Cycle,
		Description:       fmt.Sprintf("Assinatura %s - %s", plan.Name, company.Name),
		ExternalReference: orgID, // IMPORTANTE: organization_id para webhook
		CreditCardToken:   creditCardToken,
	}

	subscriptionResp, err := uc.asaasService.CreateSubscription(ctx, subscriptionReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create subscription in Asaas", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}

	// 7. Update company with subscription info
	company.AsaasSubscriptionID = &subscriptionResp.ID
	company.SubscriptionPlan = plan.Name
	company.SubscriptionStatus = "active" // Will be updated by webhook
	now := time.Now()
	company.SubscriptionStartedAt = &now

	// Parse next due date
	nextDue, err := time.Parse("2006-01-02", subscriptionResp.NextDueDate)
	if err == nil {
		company.NextPaymentDue = &nextDue
	}

	if err := uc.companyRepo.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company with subscription", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		// Continue anyway, subscription was created in Asaas
	}

	uc.logger.Info(ctx, "Subscription created successfully", map[string]interface{}{
		"organization_id":   orgID,
		"subscription_id":   subscriptionResp.ID,
		"plan_name":         plan.Name,
	})

	c.JSON(http.StatusCreated, SubscribeResponse{
		SubscriptionID: subscriptionResp.ID,
		Status:         subscriptionResp.Status,
		PlanName:       plan.Name,
		Value:          plan.Price,
		Cycle:          plan.Cycle,
		NextDueDate:    subscriptionResp.NextDueDate,
	})
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

	// Get company
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find company"})
		return
	}

	if company == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	if company.AsaasSubscriptionID == nil || *company.AsaasSubscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active subscription to cancel"})
		return
	}

	// Cancel in Asaas
	if err := uc.asaasService.CancelSubscription(ctx, *company.AsaasSubscriptionID); err != nil {
		uc.logger.Error(ctx, "Failed to cancel subscription in Asaas", map[string]interface{}{
			"error":           err.Error(),
			"subscription_id": *company.AsaasSubscriptionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel subscription"})
		return
	}

	// Update company
	company.SubscriptionStatus = "cancelled"
	emptyStr := ""
	company.AsaasSubscriptionID = &emptyStr

	if err := uc.companyRepo.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company after cancellation", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		// Continue anyway
	}

	uc.logger.Info(ctx, "Subscription cancelled successfully", map[string]interface{}{
		"organization_id": orgID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled successfully"})
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

	// Get company
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find company"})
		return
	}

	if company == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Company not found"})
		return
	}

	if company.AsaasSubscriptionID == nil || *company.AsaasSubscriptionID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active subscription"})
		return
	}

	// Get subscription from Asaas
	subscription, err := uc.asaasService.GetSubscription(ctx, *company.AsaasSubscriptionID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get subscription from Asaas", map[string]interface{}{
			"error":           err.Error(),
			"subscription_id": *company.AsaasSubscriptionID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get subscription status"})
		return
	}

	c.JSON(http.StatusOK, SubscribeResponse{
		SubscriptionID: subscription.ID,
		Status:         subscription.Status,
		PlanName:       company.SubscriptionPlan,
		Value:          subscription.Value,
		Cycle:          subscription.Cycle,
		NextDueDate:    subscription.NextDueDate,
	})
}

// Helper function
func (uc *ManageSubscriptionUseCase) createAsaasCustomer(ctx context.Context, company *companyEntities.CompanyEntity, orgID string) (string, error) {
	customerReq := services.AsaasCustomerRequest{
		Name:              company.Name,
		Email:             stringPtrValue(company.Email),
		CpfCnpj:           stringPtrValue(company.Document),
		Phone:             stringPtrValue(company.Phone),
		ExternalReference: orgID,
	}

	customerResp, err := uc.asaasService.CreateCustomer(ctx, customerReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Asaas customer", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		return "", err
	}

	return customerResp.ID, nil
}

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
