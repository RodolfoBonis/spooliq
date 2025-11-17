package usecases

import (
	"context"
	"fmt"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
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

	// Check if transition is valid
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

	// Save status history
	history := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       budget.ID,
		PreviousStatus: budget.Status,
		NewStatus:      request.Status,
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
	budget.Status = request.Status
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
	response, _ := buildBudgetResponse(ctx, uc.budgetRepository, budgetID, organizationID)

	uc.logger.Info(ctx, "Budget status updated successfully", map[string]interface{}{
		"budget_id":  budget.ID,
		"new_status": budget.Status,
	})

	c.JSON(http.StatusOK, response)
}

// buildBudgetResponse builds a complete budget response with items and filaments (helper function)
func buildBudgetResponse(ctx context.Context, repo repositories.BudgetRepository, budgetID uuid.UUID, organizationID string) (*entities.BudgetResponse, error) {
	budget, err := repo.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		return nil, err
	}

	customerInfo, _ := repo.GetCustomerInfo(ctx, budget.CustomerID)
	items, _ := repo.GetItems(ctx, budget.ID)

	itemResponses := make([]entities.BudgetItemResponse, len(items))
	var totalPrintMinutes int

	for i, item := range items {
		// Get filament usage info for this item
		filaments, _ := repo.GetFilamentUsageInfo(ctx, item.ID)

		// Calculate print time display
		printTimeDisplay := ""
		if item.PrintTimeHours > 0 {
			printTimeDisplay = fmt.Sprintf("%dh%02dm", item.PrintTimeHours, item.PrintTimeMinutes)
		} else {
			printTimeDisplay = fmt.Sprintf("%dm", item.PrintTimeMinutes)
		}

		// Sum total print time
		totalPrintMinutes += (item.PrintTimeHours * 60) + item.PrintTimeMinutes

		// Convert CostPresetID to string pointer
		var costPresetIDStr *string
		if item.CostPresetID != nil {
			s := item.CostPresetID.String()
			costPresetIDStr = &s
		}

		itemResponses[i] = entities.BudgetItemResponse{
			ID:                      item.ID.String(),
			BudgetID:                item.BudgetID.String(),
			ProductName:             item.ProductName,
			ProductDescription:      item.ProductDescription,
			ProductQuantity:         item.ProductQuantity,
			ProductDimensions:       item.ProductDimensions,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			PrintTimeDisplay:        printTimeDisplay,
			CostPresetID:            costPresetIDStr,
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			AdditionalNotes:     item.AdditionalNotes,
			FilamentCost:            item.FilamentCost,
			WasteCost:               item.WasteCost,
			EnergyCost:              item.EnergyCost,
			SetupCost:               item.SetupCost,
			ManualLaborCost:         item.ManualLaborCost,
			ItemTotalCost:           item.ItemTotalCost,
			UnitPrice:               item.UnitPrice,
			Filaments:           filaments,
			Order:               item.Order,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		}
	}

	// Calculate total print time
	totalHours := totalPrintMinutes / 60
	totalMins := totalPrintMinutes % 60
	totalPrintTimeDisplay := ""
	if totalHours > 0 {
		totalPrintTimeDisplay = fmt.Sprintf("%dh%02dm", totalHours, totalMins)
	} else {
		totalPrintTimeDisplay = fmt.Sprintf("%dm", totalMins)
	}

	return &entities.BudgetResponse{
		BudgetEntity:          budget,
		Customer:              customerInfo,
		Items:                 itemResponses,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: totalPrintTimeDisplay,
	}, nil
}
