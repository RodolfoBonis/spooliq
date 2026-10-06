package usecases

import (
	"errors"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UpdateStatus updates the status of a budget
// @Summary Update budget status
// @Description Update the status of a budget with validation
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param request body entities.UpdateStatusRequest true "Update status request"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /budgets/{id}/status [patch]
// @Security BearerAuth
func (uc *BudgetUseCase) UpdateStatus(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID required"})
		return
	}

	uc.logger.Info(ctx, "Budget status update attempt started", map[string]interface{}{
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

	var request entities.UpdateStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind request", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid request format")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Validate request
	if err := uc.validator.Struct(request); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Validation failed: " + err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get existing budget
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Check if transition is valid (allowed-transition validation is preserved).
	if !budget.IsValidTransition(request.Status) {
		uc.logger.Error(ctx, "Invalid status transition", map[string]interface{}{
			"budget_id":        budgetID,
			"current_status":   budget.Status,
			"requested_status": request.Status,
		})
		appError := coreErrors.UsecaseError("Invalid status transition")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Capture the status we transitioned from; it is the optimistic guard for the
	// write (UpdateStatus only matches rows still in this status).
	previousStatus := budget.Status

	history := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       budget.ID,
		OrganizationID: organizationID,
		PreviousStatus: previousStatus,
		NewStatus:      request.Status,
		ChangedBy:      userID,
		Notes:          request.Notes,
		CreatedAt:      time.Now(),
	}

	// Do the status update and the history insert atomically in ONE transaction so
	// a history row is never recorded for a transition that didn't actually apply
	// (and vice-versa). The status write is guarded by the expected current status;
	// if a concurrent change already moved the budget off previousStatus the write
	// matches no row and we surface a 409 Conflict.
	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.UpdateStatus(ctx, budget.ID, organizationID, previousStatus, request.Status); err != nil {
			return err
		}
		return repo.AddStatusHistory(ctx, history)
	}); err != nil {
		if errors.Is(err, entities.ErrBudgetStatusConflict) {
			uc.logger.Warning(ctx, "Budget status changed concurrently", map[string]interface{}{
				"budget_id":       budgetID,
				"expected_status": previousStatus,
			})
			appError := coreErrors.ConflictError("O status do orçamento foi alterado por outra requisição. Recarregue e tente novamente.")
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		uc.logger.Error(ctx, "Failed to update budget status", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response from the freshly stored state.
	response, _ := uc.buildBudgetResponse(ctx, budgetID, organizationID)

	uc.logger.Info(ctx, "Budget status updated successfully", map[string]interface{}{
		"budget_id":  budget.ID,
		"new_status": request.Status,
	})

	c.JSON(http.StatusOK, response)

	// Record activity (fire-and-forget)
	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionStatusChanged,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budget.ID.String(),
		EntityName:     budget.Name,
		Metadata: map[string]any{
			"previous_status": string(previousStatus),
			"new_status":      string(request.Status),
		},
		CreatedAt: time.Now(),
	})
}
