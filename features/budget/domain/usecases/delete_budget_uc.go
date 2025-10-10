package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Delete deletes a budget (soft delete, only if not printing/completed)
// @Summary Delete budget
// @Description Delete a budget (soft delete)
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id} [delete]
// @Security BearerAuth
func (uc *BudgetUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "Budget deletion attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Parse budget ID
	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid budget ID", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid budget ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Check if budget exists and user has permission
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Check if budget can be deleted
	if !budget.CanBeDeleted() {
		uc.logger.Error(ctx, "Cannot delete printing or completed budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		appError := coreErrors.UsecaseError("Cannot delete printing or completed budgets")
		c.JSON(http.StatusConflict, gin.H{"error": appError.Message})
		return
	}

	// Delete budget
	if err := uc.budgetRepository.Delete(ctx, budgetID); err != nil {
		uc.logger.Error(ctx, "Failed to delete budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Budget deleted successfully", map[string]interface{}{
		"budget_id": budgetID,
	})

	c.Status(http.StatusNoContent)
}
