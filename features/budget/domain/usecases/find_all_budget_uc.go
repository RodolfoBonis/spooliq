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

		// Get items
		items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)
		itemResponses := make([]entities.BudgetItemResponse, len(items))
		for j, item := range items {
			filamentInfo, _ := uc.budgetRepository.GetFilamentInfo(ctx, item.FilamentID)
			itemResponses[j] = entities.BudgetItemResponse{
				ID:                 item.ID.String(),
				BudgetID:           item.BudgetID.String(),
				FilamentID:         item.FilamentID.String(),
				Filament:           filamentInfo,
				Quantity:           item.Quantity,
				Order:              item.Order,
				WasteAmount:        item.WasteAmount,
				ItemCost:           item.ItemCost,
				ProductName:        item.ProductName,
				ProductDescription: item.ProductDescription,
				ProductQuantity:    item.ProductQuantity,
				UnitPrice:          item.UnitPrice,
				ProductDimensions:  item.ProductDimensions,
				CreatedAt:          item.CreatedAt,
				UpdatedAt:          item.UpdatedAt,
			}
		}

		budgetResponses[i] = entities.BudgetResponse{
			Budget:   budget,
			Customer: customerInfo,
			Items:    itemResponses,
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
