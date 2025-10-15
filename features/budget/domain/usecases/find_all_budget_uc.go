package usecases

import (
	"fmt"
	"net/http"
	"strconv"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
)

// FindAll retrieves all budgets with pagination
// @Summary List budgets
// @Description Get all budgets with pagination
// @Tags budgets
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} entities.ListBudgetsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "Budgets retrieval attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get budgets from repository
	budgets, total, err := uc.budgetRepository.FindAll(ctx, organizationID, pageSize, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budgets", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response
	budgetResponses := make([]entities.BudgetResponse, len(budgets))
	for i, budget := range budgets {
		// Get customer info
		customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)

		// Get items with filaments
		items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)
		itemResponses := make([]entities.BudgetItemResponse, len(items))
		var totalPrintMinutes int

		for j, item := range items {
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

			itemResponses[j] = entities.BudgetItemResponse{
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

		budgetResponses[i] = entities.BudgetResponse{
			Budget:                budget,
			Customer:              customerInfo,
			Items:                 itemResponses,
			TotalPrintTimeHours:   totalHours,
			TotalPrintTimeMinutes: totalMins,
			TotalPrintTimeDisplay: totalPrintTimeDisplay,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := entities.ListBudgetsResponse{
		Data:       budgetResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	uc.logger.Info(ctx, "Budgets retrieved successfully", map[string]interface{}{
		"count": len(budgets),
		"total": total,
		"page":  page,
	})

	c.JSON(http.StatusOK, response)
}
