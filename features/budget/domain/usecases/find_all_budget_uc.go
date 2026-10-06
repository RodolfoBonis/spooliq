package usecases

import (
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
// @Router /budgets [get]
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

	// Build response using the shared builder so every budget carries the same
	// per-item sale values and cost-preset references as the detail endpoints.
	budgetResponses := make([]entities.BudgetResponse, len(budgets))
	for i, budget := range budgets {
		customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
		items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)

		itemResponses, totalHours, totalMins := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.TotalCost, organizationID)

		budgetResponses[i] = entities.BudgetResponse{
			BudgetEntity:          budget,
			Customer:              customerInfo,
			Items:                 itemResponses,
			TotalPrintTimeHours:   totalHours,
			TotalPrintTimeMinutes: totalMins,
			TotalPrintTimeDisplay: formatPrintTime(totalHours, totalMins),
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
