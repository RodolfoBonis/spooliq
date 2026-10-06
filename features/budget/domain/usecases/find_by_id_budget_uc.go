package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindByID retrieves a budget by ID
// @Summary Get budget by ID
// @Description Get a specific budget by ID with all details
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id} [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	// Parse budget ID
	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid budget ID", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	// Build the full response (budget + customer + items with sale values).
	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		respondBudgetError(c, err)
		return
	}

	uc.logger.Info(ctx, "Budget retrieved successfully", map[string]interface{}{
		"budget_id": budgetID,
	})

	c.JSON(http.StatusOK, response)
}
