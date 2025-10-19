package plans

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UpdatePlanRequest represents the request to update a plan
type UpdatePlanRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"min=0"`
	Currency    string  `json:"currency"`
	Interval    string  `json:"interval" binding:"oneof=MONTHLY YEARLY"`
	Active      *bool   `json:"active"`
	Popular     *bool   `json:"popular"`
	Recommended *bool   `json:"recommended"`
	SortOrder   *int    `json:"sort_order"`
}

// UpdatePlanUseCase handles plan updates by admin
type UpdatePlanUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewUpdatePlanUseCase creates a new instance
func NewUpdatePlanUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *UpdatePlanUseCase {
	return &UpdatePlanUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute updates an existing plan
func (uc *UpdatePlanUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	// Get plan ID from URL
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "ID de plano inválido"))
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Dados inválidos: "+err.Error()))
		return
	}

	// Get existing plan
	plan, err := uc.planRepository.FindByID(ctx, planID)
	if err != nil {
		uc.logger.Error(ctx, "Plan not found", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": planID,
		})
		c.JSON(http.StatusNotFound, errors.NewHTTPError(http.StatusNotFound, "Plano não encontrado"))
		return
	}

	// Update fields if provided
	if req.Name != "" {
		plan.Name = req.Name
	}
	if req.Description != "" {
		plan.Description = req.Description
	}
	if req.Price > 0 {
		plan.Price = req.Price
	}
	if req.Currency != "" {
		plan.Currency = req.Currency
	}
	if req.Interval != "" {
		plan.Interval = req.Interval
	}
	if req.Active != nil {
		plan.Active = *req.Active
	}
	if req.Popular != nil {
		plan.Popular = *req.Popular
	}
	if req.Recommended != nil {
		plan.Recommended = *req.Recommended
	}
	if req.SortOrder != nil {
		plan.SortOrder = *req.SortOrder
	}

	// Save to database
	if err := uc.planRepository.Update(ctx, plan); err != nil {
		uc.logger.Error(ctx, "Failed to update plan", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": planID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao atualizar plano"))
		return
	}

	uc.logger.Info(ctx, "Plan updated successfully", map[string]interface{}{
		"plan_id": planID,
	})

	c.JSON(http.StatusOK, plan)
}
