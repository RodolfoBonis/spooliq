package usecases

import (
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

	// Get items
	items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)
	itemResponses := make([]entities.BudgetItemResponse, len(items))
	for i, item := range items {
		filamentInfo, _ := uc.budgetRepository.GetFilamentInfo(ctx, item.FilamentID)
		itemResponses[i] = entities.BudgetItemResponse{
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

	// Get presets info
	var machinePreset, energyPreset, costPreset *entities.PresetInfo
	if budget.MachinePresetID != nil {
		machinePreset, _ = uc.budgetRepository.GetPresetInfo(ctx, *budget.MachinePresetID, "machine")
	}
	if budget.EnergyPresetID != nil {
		energyPreset, _ = uc.budgetRepository.GetPresetInfo(ctx, *budget.EnergyPresetID, "energy")
	}
	if budget.CostPresetID != nil {
		costPreset, _ = uc.budgetRepository.GetPresetInfo(ctx, *budget.CostPresetID, "cost")
	}

	response := entities.BudgetResponse{
		Budget:        budget,
		Customer:      customerInfo,
		Items:         itemResponses,
		MachinePreset: machinePreset,
		EnergyPreset:  energyPreset,
		CostPreset:    costPreset,
	}

	uc.logger.Info(ctx, "Budget retrieved successfully", map[string]interface{}{
		"budget_id": budget.ID,
	})

	c.JSON(http.StatusOK, response)
}
