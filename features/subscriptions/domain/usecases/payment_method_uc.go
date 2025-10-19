package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentMethodUseCase handles payment method operations
type PaymentMethodUseCase struct {
	paymentMethodRepo repositories.PaymentMethodRepository
	companyRepo       companyRepo.CompanyRepository
	asaasService      services.IAsaasService
	logger            logger.Logger
}

// NewPaymentMethodUseCase creates a new instance of PaymentMethodUseCase
func NewPaymentMethodUseCase(
	paymentMethodRepo repositories.PaymentMethodRepository,
	companyRepo companyRepo.CompanyRepository,
	asaasService services.IAsaasService,
	logger logger.Logger,
) *PaymentMethodUseCase {
	return &PaymentMethodUseCase{
		paymentMethodRepo: paymentMethodRepo,
		companyRepo:       companyRepo,
		asaasService:      asaasService,
		logger:            logger,
	}
}

// AddPaymentMethod tokenizes and saves a credit card
// @Summary Add payment method
// @Description Tokenize and save a credit card for the organization
// @Tags payment-methods
// @Accept json
// @Produce json
// @Param request body entities.PaymentMethodCreateRequest true "Payment method data"
// @Success 201 {object} entities.PaymentMethodResponse "Payment method created"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/payment-methods [post]
func (uc *PaymentMethodUseCase) AddPaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	var req entities.PaymentMethodCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid payment method request", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get company to ensure it exists and get asaas_customer_id
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

	// If company doesn't have an Asaas customer ID, create one
	asaasCustomerID := ""
	if company.AsaasCustomerID == nil || *company.AsaasCustomerID == "" {
		customerReq := services.AsaasCustomerRequest{
			Name:              company.Name,
			Email:             stringValue(company.Email),
			CpfCnpj:           stringValue(company.Document),
			Phone:             stringValue(company.Phone),
			ExternalReference: orgID,
		}

		customerResp, err := uc.asaasService.CreateCustomer(ctx, customerReq)
		if err != nil {
			uc.logger.Error(ctx, "Failed to create Asaas customer", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment customer"})
			return
		}

		asaasCustomerID = customerResp.ID
		company.AsaasCustomerID = &asaasCustomerID

		if err := uc.companyRepo.Update(ctx, company); err != nil {
			uc.logger.Error(ctx, "Failed to update company with Asaas customer ID", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": orgID,
			})
			// Continue anyway, we have the customer ID
		}
	} else {
		asaasCustomerID = *company.AsaasCustomerID
	}

	// Tokenize credit card in Asaas
	tokenReq := services.AsaasTokenizeCreditCardRequest{
		Customer: asaasCustomerID,
		CreditCard: services.AsaasCreditCardInfo{
			HolderName:  req.HolderName,
			Number:      req.Number,
			ExpiryMonth: req.ExpiryMonth,
			ExpiryYear:  req.ExpiryYear,
			Ccv:         req.Ccv,
		},
	}

	tokenResp, err := uc.asaasService.TokenizeCreditCard(ctx, tokenReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to tokenize credit card", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to tokenize credit card"})
		return
	}

	// If this should be the primary method, unset current primary
	if req.SetAsPrimary {
		// This will be handled by the repository's SetAsPrimary method
	}

	// Save payment method
	paymentMethod := &entities.PaymentMethodEntity{
		OrganizationID:       orgID,
		AsaasCreditCardToken: tokenResp.CreditCardToken,
		HolderName:           req.HolderName,
		Last4Digits:          tokenResp.CreditCardNumber, // Last 4 digits from Asaas
		Brand:                tokenResp.CreditCardBrand,
		ExpiryMonth:          req.ExpiryMonth,
		ExpiryYear:           req.ExpiryYear,
		IsPrimary:            req.SetAsPrimary,
	}

	if err := uc.paymentMethodRepo.Create(ctx, paymentMethod); err != nil {
		uc.logger.Error(ctx, "Failed to save payment method", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payment method"})
		return
	}

	// If set as primary, ensure it's the only primary
	if req.SetAsPrimary {
		if err := uc.paymentMethodRepo.SetAsPrimary(ctx, orgID, paymentMethod.ID); err != nil {
			uc.logger.Error(ctx, "Failed to set payment method as primary", map[string]interface{}{
				"error":           err.Error(),
				"payment_method_id": paymentMethod.ID,
			})
			// Continue anyway, payment method was saved
		}
	}

	uc.logger.Info(ctx, "Payment method added successfully", map[string]interface{}{
		"organization_id":   orgID,
		"payment_method_id": paymentMethod.ID,
		"is_primary":        paymentMethod.IsPrimary,
	})

	c.JSON(http.StatusCreated, toPaymentMethodResponse(paymentMethod))
}

// ListPaymentMethods lists all payment methods for the organization
// @Summary List payment methods
// @Description List all payment methods for the organization
// @Tags payment-methods
// @Produce json
// @Success 200 {array} entities.PaymentMethodResponse "Payment methods list"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/payment-methods [get]
func (uc *PaymentMethodUseCase) ListPaymentMethods(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	paymentMethods, err := uc.paymentMethodRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list payment methods", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list payment methods"})
		return
	}

	response := make([]entities.PaymentMethodResponse, len(paymentMethods))
	for i, pm := range paymentMethods {
		response[i] = *toPaymentMethodResponse(pm)
	}

	c.JSON(http.StatusOK, response)
}

// SetPrimaryPaymentMethod sets a payment method as primary
// @Summary Set primary payment method
// @Description Set a payment method as primary (and unset others)
// @Tags payment-methods
// @Param id path string true "Payment Method ID"
// @Success 200 {object} map[string]string "Payment method set as primary"
// @Failure 400 {object} map[string]string "Invalid ID"
// @Failure 404 {object} map[string]string "Payment method not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/payment-methods/{id}/set-primary [put]
func (uc *PaymentMethodUseCase) SetPrimaryPaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	// Verify payment method exists and belongs to organization
	paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find payment method"})
		return
	}

	if paymentMethod == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	if paymentMethod.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Payment method does not belong to your organization"})
		return
	}

	// Set as primary
	if err := uc.paymentMethodRepo.SetAsPrimary(ctx, orgID, id); err != nil {
		uc.logger.Error(ctx, "Failed to set payment method as primary", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set payment method as primary"})
		return
	}

	uc.logger.Info(ctx, "Payment method set as primary", map[string]interface{}{
		"organization_id":   orgID,
		"payment_method_id": id,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Payment method set as primary"})
}

// DeletePaymentMethod deletes a payment method
// @Summary Delete payment method
// @Description Soft delete a payment method
// @Tags payment-methods
// @Param id path string true "Payment Method ID"
// @Success 200 {object} map[string]string "Payment method deleted"
// @Failure 400 {object} map[string]string "Invalid ID or cannot delete primary"
// @Failure 404 {object} map[string]string "Payment method not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /v1/payment-methods/{id} [delete]
func (uc *PaymentMethodUseCase) DeletePaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment method ID"})
		return
	}

	// Verify payment method exists and belongs to organization
	paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find payment method"})
		return
	}

	if paymentMethod == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment method not found"})
		return
	}

	if paymentMethod.OrganizationID != orgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Payment method does not belong to your organization"})
		return
	}

	// Don't allow deleting primary payment method if there are others
	if paymentMethod.IsPrimary {
		allMethods, err := uc.paymentMethodRepo.FindByOrganizationID(ctx, orgID)
		if err == nil && len(allMethods) > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete primary payment method. Set another as primary first."})
			return
		}
	}

	// Delete
	if err := uc.paymentMethodRepo.Delete(ctx, id); err != nil {
		uc.logger.Error(ctx, "Failed to delete payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
		return
	}

	uc.logger.Info(ctx, "Payment method deleted", map[string]interface{}{
		"organization_id":   orgID,
		"payment_method_id": id,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted"})
}

// Helper functions
func toPaymentMethodResponse(pm *entities.PaymentMethodEntity) *entities.PaymentMethodResponse {
	return &entities.PaymentMethodResponse{
		ID:          pm.ID,
		HolderName:  pm.HolderName,
		Last4Digits: pm.Last4Digits,
		Brand:       pm.Brand,
		ExpiryMonth: pm.ExpiryMonth,
		ExpiryYear:  pm.ExpiryYear,
		IsPrimary:   pm.IsPrimary,
		CreatedAt:   pm.CreatedAt,
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
