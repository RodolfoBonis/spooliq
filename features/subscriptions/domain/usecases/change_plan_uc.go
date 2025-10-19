package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
)

// ChangePlanRequest represents the request to change subscription plan
type ChangePlanRequest struct {
	PlanSlug string `json:"plan_slug" binding:"required"`
}

// ChangePlanResponse represents the response for plan change
type ChangePlanResponse struct {
	Success  bool    `json:"success"`
	Message  string  `json:"message"`
	PlanName string  `json:"plan_name"`
	NewValue float64 `json:"new_value"`
}

// ChangePlanUseCase handles plan upgrades and downgrades
type ChangePlanUseCase struct {
	asaasService      services.IAsaasService
	companyRepository companyRepositories.CompanyRepository
	planRepository    repositories.PlanRepository
	logger            logger.Logger
}

// NewChangePlanUseCase creates a new instance
func NewChangePlanUseCase(
	asaasService services.IAsaasService,
	companyRepository companyRepositories.CompanyRepository,
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *ChangePlanUseCase {
	return &ChangePlanUseCase{
		asaasService:      asaasService,
		companyRepository: companyRepository,
		planRepository:    planRepository,
		logger:            logger,
	}
}

// Execute handles the plan change
func (uc *ChangePlanUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	var req ChangePlanRequest
	
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

	// Get new plan from database
	newPlan, err := uc.planRepository.FindBySlug(ctx, req.PlanSlug)
	if err != nil {
		uc.logger.Error(ctx, "Plan not found", map[string]interface{}{
			"error":     err.Error(),
			"plan_slug": req.PlanSlug,
		})
		c.JSON(http.StatusNotFound, errors.NewHTTPError(http.StatusNotFound, "Plano não encontrado"))
		return
	}

	// Check if plan is active
	if !newPlan.Active {
		uc.logger.Error(ctx, "Plan is not active", map[string]interface{}{
			"plan_slug": req.PlanSlug,
			"plan_name": newPlan.Name,
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Plano não está disponível para contratação"))
		return
	}

	// Check if trying to change to the same plan
	if company.SubscriptionPlan == newPlan.Slug {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Você já está no plano "+newPlan.Name))
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

	uc.logger.Info(ctx, "Plan change requested", map[string]interface{}{
		"organization_id":       organizationID,
		"current_plan":          company.SubscriptionPlan,
		"new_plan":              newPlan.Slug,
		"asaas_subscription_id": *company.AsaasSubscriptionID,
	})

	// Update subscription in Asaas with new value
	updateReq := services.AsaasSubscriptionUpdateRequest{
		Value:                 newPlan.Price,
		Description:           "SpoolIQ - " + newPlan.Name,
		UpdatePendingPayments: true, // Update pending payments with new value
	}

	_, err = uc.asaasService.UpdateSubscription(ctx, *company.AsaasSubscriptionID, updateReq)
	if err != nil {
		uc.logger.Error(ctx, "Failed to update subscription in Asaas", map[string]interface{}{
			"error":                 err.Error(),
			"asaas_subscription_id": *company.AsaasSubscriptionID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao alterar plano"))
		return
	}

	// Update company plan in database
	company.SubscriptionPlan = newPlan.Slug
	if err := uc.companyRepository.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company plan", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		// Don't return error as subscription was already updated in Asaas
	}

	uc.logger.Info(ctx, "Plan changed successfully", map[string]interface{}{
		"organization_id": organizationID,
		"new_plan":        newPlan.Slug,
		"new_value":       newPlan.Price,
	})

	response := ChangePlanResponse{
		Success:  true,
		Message:  "Plano alterado com sucesso!",
		PlanName: newPlan.Name,
		NewValue: newPlan.Price,
	}

	c.JSON(http.StatusOK, response)
}
