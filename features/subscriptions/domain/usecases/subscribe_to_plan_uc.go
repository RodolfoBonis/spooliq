package usecases

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscribeRequest represents the request to subscribe to a plan
type SubscribeRequest struct {
	PlanID          uuid.UUID  `json:"plan_id" binding:"required"`
	PaymentMethodID *uuid.UUID `json:"payment_method_id"` // Optional if using boleto/pix
	BillingType     string     `json:"billing_type" binding:"required,oneof=CREDIT_CARD BOLETO PIX"`
}

// SubscribeResponse represents the response after subscribing
type SubscribeResponse struct {
	SubscriptionID      string  `json:"subscription_id"` // Asaas subscription ID
	Status              string  `json:"status"`          // ACTIVE, PENDING, etc
	PlanName            string  `json:"plan_name"`
	Value               float64 `json:"value"`
	Cycle               string  `json:"cycle"`
	NextDueDate         string  `json:"next_due_date"`
	FirstPaymentID      *string `json:"first_payment_id,omitempty"`
	FirstPaymentInvoice *string `json:"first_payment_invoice,omitempty"`
}

// ManageSubscriptionUseCase handles subscription management operations
type ManageSubscriptionUseCase struct {
	planRepo               repositories.SubscriptionPlanRepository
	paymentMethodRepo      repositories.PaymentMethodRepository
	paymentGatewayLinkRepo repositories.PaymentGatewayLinkRepository
	companyRepo            companyRepo.CompanyRepository
	asaasService           services.IAsaasService
	logger                 logger.Logger
}

// NewManageSubscriptionUseCase creates a new instance
func NewManageSubscriptionUseCase(
	planRepo repositories.SubscriptionPlanRepository,
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentGatewayLinkRepo repositories.PaymentGatewayLinkRepository,
	companyRepo companyRepo.CompanyRepository,
	asaasService services.IAsaasService,
	logger logger.Logger,
) *ManageSubscriptionUseCase {
	return &ManageSubscriptionUseCase{
		planRepo:               planRepo,
		paymentMethodRepo:      paymentMethodRepo,
		paymentGatewayLinkRepo: paymentGatewayLinkRepo,
		companyRepo:            companyRepo,
		asaasService:           asaasService,
		logger:                 logger,
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
// @Router /subscriptions/subscribe [post]
func (uc *ManageSubscriptionUseCase) SubscribeToPlan(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid subscription request", map[string]interface{}{
			"error": err.Error(),
		})
		coreerrors.Respond(c, err)
		return
	}

	// 1. Get company to ensure it exists
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if company == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("company_not_found", "Empresa não encontrada"))
		return
	}

	// 2. Get subscription plan
	plan, err := uc.planRepo.FindByID(ctx, req.PlanID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find subscription plan", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": req.PlanID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if plan == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("subscription_plan_not_found", "Plano de assinatura não encontrado"))
		return
	}

	// 3. Query PaymentGatewayLinkRepository to get Asaas customer ID
	paymentGatewayLink, err := uc.paymentGatewayLinkRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to query PaymentGatewayLink", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	var asaasCustomerID string

	if paymentGatewayLink == nil {
		// Create Asaas customer + PaymentGatewayLink record
		asaasCustomerReq := services.AsaasCustomerRequest{
			Name:    company.Name,
			CpfCnpj: stringPtrValue(company.Document),
			Email:   stringPtrValue(company.Email),
			Phone:   stringPtrValue(company.Phone),
		}

		asaasCustomer, err := uc.asaasService.CreateCustomer(ctx, asaasCustomerReq)
		if err != nil {
			uc.logger.Error(ctx, "Failed to create Asaas customer", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
			})
			coreerrors.Respond(c, coreerrors.ExternalServiceError("Falha ao criar conta de pagamento"))
			return
		}

		// Create PaymentGatewayLink record
		newPaymentGatewayLink := &entities.PaymentGatewayLinkEntity{
			OrganizationID: orgID,
			Gateway:        "asaas",
			CustomerID:     asaasCustomer.ID,
		}

		if err := uc.paymentGatewayLinkRepo.Create(ctx, newPaymentGatewayLink); err != nil {
			uc.logger.Error(ctx, "Failed to create PaymentGatewayLink", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
				"customer_id":     asaasCustomer.ID,
			})
			coreerrors.Respond(c, err)
			return
		}

		asaasCustomerID = asaasCustomer.ID
		uc.logger.Info(ctx, "Created Asaas customer and PaymentGatewayLink for subscription", map[string]interface{}{
			"organization_id": orgID,
			"customer_id":     asaasCustomerID,
		})
	} else {
		asaasCustomerID = paymentGatewayLink.CustomerID
		uc.logger.Info(ctx, "Using existing Asaas customer for subscription", map[string]interface{}{
			"organization_id": orgID,
			"customer_id":     asaasCustomerID,
		})
	}

	// 4. Handle payment method for CREDIT_CARD
	var asaasCreditCardToken string
	if req.BillingType == "CREDIT_CARD" {
		if req.PaymentMethodID == nil {
			// Try to get primary payment method
			paymentMethods, err := uc.paymentMethodRepo.FindByOrganizationID(ctx, orgID)
			if err != nil {
				uc.logger.Error(ctx, "Failed to find payment methods", map[string]interface{}{
					"error":           err.Error(),
					"organization_id": orgID,
				})
				coreerrors.Respond(c, err)
				return
			}

			var primaryPaymentMethod *entities.PaymentMethodEntity
			for _, pm := range paymentMethods {
				if pm.IsPrimary {
					primaryPaymentMethod = pm
					break
				}
			}

			if primaryPaymentMethod == nil {
				coreerrors.Respond(c, coreerrors.BadRequest("no_payment_method", "Nenhum método de pagamento encontrado. Adicione um cartão de crédito primeiro."))
				return
			}

			asaasCreditCardToken = primaryPaymentMethod.AsaasCreditCardToken
			req.PaymentMethodID = &primaryPaymentMethod.ID
		} else {
			// Get specified payment method
			paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, *req.PaymentMethodID)
			if err != nil {
				uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
					"error":             err.Error(),
					"payment_method_id": *req.PaymentMethodID,
				})
				coreerrors.Respond(c, err)
				return
			}

			if paymentMethod == nil {
				coreerrors.Respond(c, coreerrors.NotFoundErr("payment_method_not_found", "Método de pagamento não encontrado"))
				return
			}

			if paymentMethod.OrganizationID != orgID {
				coreerrors.Respond(c, coreerrors.NotFoundErr("payment_method_not_found", "Método de pagamento não encontrado"))
				return
			}

			asaasCreditCardToken = paymentMethod.AsaasCreditCardToken
		}
	}

	// 5. Create subscription in Asaas
	nextDueDate := time.Now().AddDate(0, 1, 0) // Next month
	if plan.Cycle == "YEARLY" {
		nextDueDate = time.Now().AddDate(1, 0, 0) // Next year
	}

	subscriptionReq := services.AsaasSubscriptionRequest{
		Customer:          asaasCustomerID,
		BillingType:       req.BillingType,
		Value:             plan.Price,
		NextDueDate:       nextDueDate.Format("2006-01-02"),
		Cycle:             plan.Cycle,
		Description:       fmt.Sprintf("Assinatura %s - %s", plan.Name, company.Name),
		ExternalReference: orgID,
		CreditCardToken:   asaasCreditCardToken,
	}

	asaasSubscription, err := uc.asaasService.CreateSubscription(ctx, subscriptionReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to create Asaas subscription", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
			"customer_id":     asaasCustomerID,
		})
		coreerrors.Respond(c, err)
		return
	}

	// 6. Update company.subscription_plan_id
	company.SubscriptionPlanID = &plan.ID
	company.SubscriptionStatus = asaasSubscription.Status
	if asaasSubscription.Status == "ACTIVE" {
		now := time.Now()
		company.SubscriptionStartedAt = &now
	}

	if err := uc.companyRepo.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company subscription", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
			"plan_id":         plan.ID,
		})
		// Don't return error here - subscription was created, this is just updating local record
	}

	// 7. Update PaymentGatewayLink with subscription ID
	if paymentGatewayLink != nil {
		paymentGatewayLink.SubscriptionID = &asaasSubscription.ID
		if err := uc.paymentGatewayLinkRepo.Update(ctx, paymentGatewayLink); err != nil {
			uc.logger.Error(ctx, "Failed to update PaymentGatewayLink with subscription ID", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
				"subscription_id": asaasSubscription.ID,
			})
			// Don't return error here - subscription was created
		}
	}

	uc.logger.Info(ctx, "Subscription created successfully", map[string]interface{}{
		"organization_id": orgID,
		"subscription_id": asaasSubscription.ID,
		"plan_id":         plan.ID,
		"plan_name":       plan.Name,
		"status":          asaasSubscription.Status,
		"billing_type":    req.BillingType,
	})

	// 8. Build response
	response := SubscribeResponse{
		SubscriptionID: asaasSubscription.ID,
		Status:         asaasSubscription.Status,
		PlanName:       plan.Name,
		Value:          plan.Price,
		Cycle:          plan.Cycle,
		NextDueDate:    asaasSubscription.NextDueDate,
	}

	c.JSON(http.StatusCreated, response)
}

// CancelSubscriptionRequest is the optional cancellation feedback.
type CancelSubscriptionRequest struct {
	Reason   string `json:"reason"`
	Feedback string `json:"feedback"`
}

// CancelSubscription cancels the current subscription
// @Summary Cancel subscription
// @Description Cancel the current active subscription. Cancellation is immediate. The optional body records the reason for churn analysis.
// @Tags subscriptions
// @Accept json
// @Produce json
// @Param request body CancelSubscriptionRequest false "Cancellation reason and feedback"
// @Success 200 {object} map[string]string "Subscription cancelled"
// @Failure 400 {object} map[string]string "No active subscription"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /subscriptions/cancel [delete]
func (uc *ManageSubscriptionUseCase) CancelSubscription(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	// The body is optional (older clients send none); ignore bind errors.
	var feedback CancelSubscriptionRequest
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&feedback)
	}

	// 1. Get company to check subscription
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if company == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("company_not_found", "Empresa não encontrada"))
		return
	}

	if company.SubscriptionPlanID == nil {
		coreerrors.Respond(c, coreerrors.BadRequest("no_active_subscription", "Nenhuma assinatura ativa encontrada"))
		return
	}

	// 2. Get PaymentGatewayLink to find subscription ID
	paymentGatewayLink, err := uc.paymentGatewayLinkRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to query PaymentGatewayLink", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if paymentGatewayLink == nil || paymentGatewayLink.SubscriptionID == nil {
		coreerrors.Respond(c, coreerrors.BadRequest("no_gateway_subscription", "Nenhuma assinatura encontrada no gateway de pagamento"))
		return
	}

	// 3. Cancel subscription in Asaas
	if err := uc.asaasService.CancelSubscription(ctx, *paymentGatewayLink.SubscriptionID); err != nil {
		uc.logger.Error(ctx, "Failed to cancel Asaas subscription", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
			"subscription_id": *paymentGatewayLink.SubscriptionID,
		})
		coreerrors.Respond(c, err)
		return
	}

	// 4. Update company record
	company.SubscriptionPlanID = nil
	company.SubscriptionStatus = "CANCELLED"
	if err := uc.companyRepo.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company subscription status", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		// Don't return error - subscription was cancelled in Asaas
	}

	// 5. Clear subscription ID from PaymentGatewayLink
	paymentGatewayLink.SubscriptionID = nil
	if err := uc.paymentGatewayLinkRepo.Update(ctx, paymentGatewayLink); err != nil {
		uc.logger.Error(ctx, "Failed to update PaymentGatewayLink", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		// Don't return error - subscription was cancelled
	}

	var subscriptionID string
	if paymentGatewayLink.SubscriptionID != nil {
		subscriptionID = *paymentGatewayLink.SubscriptionID
	}

	uc.logger.Info(ctx, "Subscription cancelled successfully", map[string]interface{}{
		"organization_id": orgID,
		"subscription_id": subscriptionID,
		"cancel_reason":   truncate(feedback.Reason, 64),
		"cancel_feedback": truncate(feedback.Feedback, 1000),
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
// @Router /subscriptions/status [get]
func (uc *ManageSubscriptionUseCase) GetSubscriptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	// 1. Get company to check subscription
	company, err := uc.companyRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if company == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("company_not_found", "Empresa não encontrada"))
		return
	}

	if company.SubscriptionPlanID == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("no_active_subscription", "Nenhuma assinatura ativa encontrada"))
		return
	}

	// 2. Get subscription plan details
	plan, err := uc.planRepo.FindByID(ctx, *company.SubscriptionPlanID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find subscription plan", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": *company.SubscriptionPlanID,
		})
		coreerrors.Respond(c, err)
		return
	}

	if plan == nil {
		coreerrors.Respond(c, coreerrors.NotFoundErr("subscription_plan_not_found", "Plano de assinatura não encontrado"))
		return
	}

	// 3. Get PaymentGatewayLink to find subscription ID
	paymentGatewayLink, err := uc.paymentGatewayLinkRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to query PaymentGatewayLink", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	var response SubscribeResponse
	response.PlanName = plan.Name
	response.Value = plan.Price
	response.Cycle = plan.Cycle
	response.Status = company.SubscriptionStatus

	// 4. If we have Asaas subscription, get current details
	if paymentGatewayLink != nil && paymentGatewayLink.SubscriptionID != nil {
		asaasSubscription, err := uc.asaasService.GetSubscription(ctx, *paymentGatewayLink.SubscriptionID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to get Asaas subscription", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
				"subscription_id": *paymentGatewayLink.SubscriptionID,
			})
			// Continue with local data only
		} else {
			response.SubscriptionID = asaasSubscription.ID
			response.Status = asaasSubscription.Status
			response.NextDueDate = asaasSubscription.NextDueDate
			response.Value = asaasSubscription.Value

			// Update local company status if different
			if company.SubscriptionStatus != asaasSubscription.Status {
				company.SubscriptionStatus = asaasSubscription.Status
				if err := uc.companyRepo.Update(ctx, company); err != nil {
					uc.logger.Error(ctx, "Failed to update company subscription status", map[string]interface{}{
						"error":           err.Error(),
						"organization_id": orgID,
					})
				}
			}
		}
	}

	uc.logger.Info(ctx, "Subscription status retrieved successfully", map[string]interface{}{
		"organization_id": orgID,
		"plan_name":       response.PlanName,
		"status":          response.Status,
	})

	c.JSON(http.StatusOK, response)
}

// truncate caps free-text fields before logging them.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
