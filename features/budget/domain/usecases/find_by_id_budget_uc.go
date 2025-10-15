package usecases

import (
	"fmt"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
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
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id} [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "Budget retrieval by ID attempt started", map[string]interface{}{
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

	// Get budget from repository
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Get customer info
	customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)

	// Get items with filaments
	items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)
	itemResponses := make([]entities.BudgetItemResponse, len(items))
	var totalPrintMinutes int

	for i, item := range items {
		// Get filament usage info for this item
		filaments, _ := uc.budgetRepository.GetFilamentUsageInfo(ctx, item.ID)

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
			ID:                  item.ID.String(),
			BudgetID:            item.BudgetID.String(),
			ProductName:         item.ProductName,
			ProductDescription:  item.ProductDescription,
			ProductQuantity:     item.ProductQuantity,
			ProductDimensions:   item.ProductDimensions,
			PrintTimeHours:      item.PrintTimeHours,
			PrintTimeMinutes:    item.PrintTimeMinutes,
			PrintTimeDisplay:    printTimeDisplay,
			CostPresetID:        costPresetIDStr,
			AdditionalLaborCost: item.AdditionalLaborCost,
			AdditionalNotes:     item.AdditionalNotes,
			FilamentCost:        item.FilamentCost,
			WasteCost:           item.WasteCost,
			EnergyCost:          item.EnergyCost,
			LaborCost:           item.LaborCost,
			ItemTotalCost:       item.ItemTotalCost,
			UnitPrice:           item.UnitPrice,
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

	response := entities.BudgetResponse{
		Budget:                budget,
		Customer:              customerInfo,
		Items:                 itemResponses,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: totalPrintTimeDisplay,
	}

	uc.logger.Info(ctx, "Budget retrieved successfully", map[string]interface{}{
		"budget_id": budget.ID,
	})

	c.JSON(http.StatusOK, response)
}
