package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/gin-gonic/gin"
)

// CancelSubscriptionRequest represents the request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CancelSubscriptionResponse represents the response for subscription cancellation
type CancelSubscriptionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CancelSubscriptionUseCase handles subscription cancellation
type CancelSubscriptionUseCase struct {
	asaasService      services.IAsaasService
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewCancelSubscriptionUseCase creates a new instance
func NewCancelSubscriptionUseCase(
	asaasService services.IAsaasService,
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *CancelSubscriptionUseCase {
	return &CancelSubscriptionUseCase{
		asaasService:      asaasService,
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// Execute handles the subscription cancellation
func (uc *CancelSubscriptionUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Dados inválidos: "+err.Error()))
		return
	}

	// Get organization from context
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusUnauthorized, errors.NewHTTPError(http.StatusUnauthorized, "Organização não encontrada"))
		return
	}

	// Get company from database
	company, err := uc.companyRepository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get company from database", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar empresa"))
		return
	}

	// Check if company has a subscription in Asaas
	if company.AsaasSubscriptionID == nil || *company.AsaasSubscriptionID == "" {
		uc.logger.Error(ctx, "Company does not have an active Asaas subscription", map[string]interface{}{
			"organization_id": organizationID,
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Empresa não possui assinatura ativa"))
		return
	}

	uc.logger.Info(ctx, "Subscription cancellation requested", map[string]interface{}{
		"organization_id":       organizationID,
		"asaas_subscription_id": *company.AsaasSubscriptionID,
		"reason":                req.Reason,
	})

	// Cancel subscription in Asaas
	if err := uc.asaasService.CancelSubscription(ctx, *company.AsaasSubscriptionID); err != nil {
		uc.logger.Error(ctx, "Failed to cancel subscription in Asaas", map[string]interface{}{
			"error":                 err.Error(),
			"asaas_subscription_id": *company.AsaasSubscriptionID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao cancelar assinatura"))
		return
	}

	// Update company status in database
	company.SubscriptionStatus = "canceled"
	if err := uc.companyRepository.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company status", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		// Don't return error as subscription was already canceled in Asaas
	}

	uc.logger.Info(ctx, "Subscription canceled successfully", map[string]interface{}{
		"organization_id": organizationID,
	})

	response := CancelSubscriptionResponse{
		Success: true,
		Message: "Assinatura cancelada com sucesso. Você terá acesso até o fim do período atual.",
	}

	c.JSON(http.StatusOK, response)
}
