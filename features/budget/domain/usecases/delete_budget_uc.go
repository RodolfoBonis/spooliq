package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Delete deletes a budget (soft delete, only if not printing/completed and unpaid)
// @Summary Delete budget
// @Description Delete a budget (soft delete)
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 204
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id} [delete]
// @Security BearerAuth
func (uc *BudgetUseCase) Delete(c *gin.Context) {
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

	// Check if budget exists and user has permission
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		respondBudgetError(c, err)
		return
	}

	// A budget that received any payment can never be deleted: money changed hands,
	// so the record must be preserved (409 budget_has_payments).
	if budget.HasPayments() {
		uc.logger.Error(ctx, "Cannot delete a budget with payments", map[string]interface{}{
			"budget_id": budgetID,
		})
		coreErrors.Respond(c, coreErrors.Conflict("budget_has_payments", "Não é possível excluir um orçamento que já recebeu pagamento"))
		return
	}

	// Check if budget can be deleted (printing/completed budgets are not deletable).
	if !budget.CanBeDeleted() {
		uc.logger.Error(ctx, "Cannot delete printing or completed budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		coreErrors.Respond(c, coreErrors.Conflict(CodeBudgetNotDeletable, "Não é possível excluir orçamentos em impressão ou concluídos"))
		return
	}

	// Delete budget
	if err := uc.budgetRepository.Delete(ctx, budgetID, organizationID); err != nil {
		uc.logger.Error(ctx, "Failed to delete budget", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	uc.logger.Info(ctx, "Budget deleted successfully", map[string]interface{}{
		"budget_id": budgetID,
	})

	c.Status(http.StatusNoContent)

	// Record activity (fire-and-forget)
	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budgetID.String(),
		EntityName:     budget.Name,
		CreatedAt:      time.Now(),
	})
}
