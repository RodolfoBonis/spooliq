package usecases

import (
	"net/http"

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

// PaymentMethodUseCase handles payment method operations
type PaymentMethodUseCase struct {
	paymentMethodRepo      repositories.PaymentMethodRepository
	paymentGatewayLinkRepo repositories.PaymentGatewayLinkRepository
	companyRepo            companyRepo.CompanyRepository
	asaasService           services.IAsaasService
	logger                 logger.Logger
}

// NewPaymentMethodUseCase creates a new instance of PaymentMethodUseCase
func NewPaymentMethodUseCase(
	paymentMethodRepo repositories.PaymentMethodRepository,
	paymentGatewayLinkRepo repositories.PaymentGatewayLinkRepository,
	companyRepo companyRepo.CompanyRepository,
	asaasService services.IAsaasService,
	logger logger.Logger,
) *PaymentMethodUseCase {
	return &PaymentMethodUseCase{
		paymentMethodRepo:      paymentMethodRepo,
		paymentGatewayLinkRepo: paymentGatewayLinkRepo,
		companyRepo:            companyRepo,
		asaasService:           asaasService,
		logger:                 logger,
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
// @Router /payment-methods [post]
func (uc *PaymentMethodUseCase) AddPaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	var req entities.PaymentMethodCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid payment method request", map[string]interface{}{
			"error": err.Error(),
		})
		coreerrors.Respond(c, err)
		return
	}

	// Get company to ensure it exists
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

	// Query PaymentGatewayLinkRepository to get/create Asaas customer ID
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
		// Create Asaas customer first
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
		uc.logger.Info(ctx, "Created Asaas customer and PaymentGatewayLink for payment method", map[string]interface{}{
			"organization_id": orgID,
			"customer_id":     asaasCustomerID,
		})
	} else {
		asaasCustomerID = paymentGatewayLink.CustomerID
		uc.logger.Info(ctx, "Using existing Asaas customer for payment method", map[string]interface{}{
			"organization_id": orgID,
			"customer_id":     asaasCustomerID,
		})
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
		coreerrors.Respond(c, coreerrors.ExternalServiceError("Falha ao tokenizar o cartão de crédito"))
		return
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
		coreerrors.Respond(c, err)
		return
	}

	// If set as primary, ensure it's the only primary
	if req.SetAsPrimary {
		if err := uc.paymentMethodRepo.SetAsPrimary(ctx, orgID, paymentMethod.ID); err != nil {
			uc.logger.Error(ctx, "Failed to set payment method as primary", map[string]interface{}{
				"error":             err.Error(),
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
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 20, max 100)"
// @Success 200 {object} helpers.Page[entities.PaymentMethodResponse] "Paginated payment methods"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Security BearerAuth
// @Router /payment-methods [get]
func (uc *PaymentMethodUseCase) ListPaymentMethods(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)

	paymentMethods, err := uc.paymentMethodRepo.FindByOrganizationID(ctx, orgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list payment methods", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": orgID,
		})
		coreerrors.Respond(c, err)
		return
	}

	response := make([]entities.PaymentMethodResponse, len(paymentMethods))
	for i, pm := range paymentMethods {
		response[i] = *toPaymentMethodResponse(pm)
	}

	// Payment methods are a small, org-scoped set; paginate in memory so the
	// response still uses the standard envelope.
	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: 20})
	total := int64(len(response))
	off := q.Offset()
	if off > len(response) {
		off = len(response)
	}
	end := len(response)
	if q.Limit() > 0 {
		end = off + q.Limit()
		if end > len(response) {
			end = len(response)
		}
	}
	c.JSON(http.StatusOK, helpers.NewPage(response[off:end], total, q))
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
// @Router /payment-methods/{id}/set-primary [put]
func (uc *PaymentMethodUseCase) SetPrimaryPaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_payment_method_id", "ID de método de pagamento inválido"))
		return
	}

	// Verify payment method exists and belongs to organization
	paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
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

	// Set as primary
	if err := uc.paymentMethodRepo.SetAsPrimary(ctx, orgID, id); err != nil {
		uc.logger.Error(ctx, "Failed to set payment method as primary", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		coreerrors.Respond(c, err)
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
// @Router /payment-methods/{id} [delete]
func (uc *PaymentMethodUseCase) DeletePaymentMethod(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := helpers.GetOrganizationIDString(c)
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_payment_method_id", "ID de método de pagamento inválido"))
		return
	}

	// Verify payment method exists and belongs to organization
	paymentMethod, err := uc.paymentMethodRepo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
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

	// Don't allow deleting primary payment method if there are others
	if paymentMethod.IsPrimary {
		allMethods, err := uc.paymentMethodRepo.FindByOrganizationID(ctx, orgID)
		if err == nil && len(allMethods) > 1 {
			coreerrors.Respond(c, coreerrors.BadRequest("cannot_delete_primary_payment_method", "Não é possível excluir o método de pagamento principal. Defina outro como principal primeiro."))
			return
		}
	}

	// Delete
	if err := uc.paymentMethodRepo.Delete(ctx, id); err != nil {
		uc.logger.Error(ctx, "Failed to delete payment method", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		coreerrors.Respond(c, err)
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

func stringPtrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
