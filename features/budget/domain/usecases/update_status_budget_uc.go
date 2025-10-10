package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
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
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/status [patch]
// @Security BearerAuth
func (uc *BudgetUseCase) UpdateStatus(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Budget status update attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	userID := getUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		appError := coreErrors.UsecaseError("User ID not found in context")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	admin := isAdmin(c)

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
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, userID, admin)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Check if transition is valid
	if !budget.IsValidTransition(request.NewStatus) {
		uc.logger.Error(ctx, "Invalid status transition", map[string]interface{}{
			"budget_id":        budgetID,
			"current_status":   budget.Status,
			"requested_status": request.NewStatus,
		})
		appError := coreErrors.UsecaseError("Invalid status transition")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Save status history
	history := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       budget.ID,
		PreviousStatus: budget.Status,
		NewStatus:      request.NewStatus,
		ChangedBy:      userID,
		Notes:          request.Notes,
		CreatedAt:      time.Now(),
	}

	if err := uc.budgetRepository.AddStatusHistory(ctx, history); err != nil {
		uc.logger.Error(ctx, "Failed to save status history", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Update budget status
	budget.Status = request.NewStatus
	budget.UpdatedAt = time.Now()

	if err := uc.budgetRepository.Update(ctx, budget); err != nil {
		uc.logger.Error(ctx, "Failed to update budget status", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response
	customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)
	items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)

	itemResponses := make([]entities.BudgetItemResponse, len(items))
	for i, item := range items {
		filamentInfo, _ := uc.budgetRepository.GetFilamentInfo(ctx, item.FilamentID)
		itemResponses[i] = entities.BudgetItemResponse{
			ID:          item.ID.String(),
			BudgetID:    item.BudgetID.String(),
			FilamentID:  item.FilamentID.String(),
			Filament:    filamentInfo,
			Quantity:    item.Quantity,
			Order:       item.Order,
			WasteAmount: item.WasteAmount,
			ItemCost:    item.ItemCost,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
	}

	response := entities.BudgetResponse{
		Budget:   budget,
		Customer: customerInfo,
		Items:    itemResponses,
	}

	uc.logger.Info(ctx, "Budget status updated successfully", map[string]interface{}{
		"budget_id":  budget.ID,
		"new_status": budget.Status,
	})

	c.JSON(http.StatusOK, response)
}

