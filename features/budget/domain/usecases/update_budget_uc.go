package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Update updates an existing budget (only draft budgets can be fully edited)
// @Summary Update budget
// @Description Update an existing budget (only drafts)
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param request body entities.UpdateBudgetRequest true "Update budget request"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id} [put]
// @Security BearerAuth
func (uc *BudgetUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "Budget update attempt started", map[string]interface{}{
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

	var request entities.UpdateBudgetRequest
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

	// Check if budget can be edited
	if !budget.CanBeEdited() {
		uc.logger.Error(ctx, "Cannot edit non-draft budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		appError := coreErrors.UsecaseError("Only draft budgets can be edited")
		c.JSON(http.StatusForbidden, gin.H{"error": appError.Message})
		return
	}

	// Update fields
	if request.Name != nil {
		budget.Name = *request.Name
	}
	if request.Description != nil {
		budget.Description = *request.Description
	}
	if request.CustomerID != nil {
		// Verify customer exists and user has permission
		_, err = uc.customerRepository.FindByID(ctx, *request.CustomerID, organizationID)
		if err != nil {
			uc.logger.Error(ctx, "Customer not found", map[string]interface{}{
				"error":       err.Error(),
				"customer_id": *request.CustomerID,
			})
			appError := coreErrors.UsecaseError("Customer not found")
			c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
			return
		}
		budget.CustomerID = *request.CustomerID
	}
	if request.PrintTimeHours != nil {
		budget.PrintTimeHours = *request.PrintTimeHours
	}
	if request.PrintTimeMinutes != nil {
		budget.PrintTimeMinutes = *request.PrintTimeMinutes
	}
	if request.MachinePresetID != nil {
		budget.MachinePresetID = request.MachinePresetID
	}
	if request.EnergyPresetID != nil {
		budget.EnergyPresetID = request.EnergyPresetID
	}
	if request.CostPresetID != nil {
		budget.CostPresetID = request.CostPresetID
	}
	if request.IncludeEnergyCost != nil {
		budget.IncludeEnergyCost = *request.IncludeEnergyCost
	}
	if request.IncludeLaborCost != nil {
		budget.IncludeLaborCost = *request.IncludeLaborCost
	}
	if request.IncludeWasteCost != nil {
		budget.IncludeWasteCost = *request.IncludeWasteCost
	}
	if request.LaborCostPerHour != nil {
		budget.LaborCostPerHour = request.LaborCostPerHour
	}
	if request.DeliveryDays != nil {
		budget.DeliveryDays = request.DeliveryDays
	}
	if request.PaymentTerms != nil {
		budget.PaymentTerms = request.PaymentTerms
	}
	if request.Notes != nil {
		budget.Notes = request.Notes
	}

	budget.UpdatedAt = time.Now()

	// Update items if provided
	if request.Items != nil {
		// Delete all existing items
		if err := uc.budgetRepository.DeleteAllItems(ctx, budgetID); err != nil {
			uc.logger.Error(ctx, "Failed to delete existing items", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Create new items
		for _, itemReq := range *request.Items {
			item := &entities.BudgetItemEntity{
				ID:         uuid.New(),
				BudgetID:   budget.ID,
				FilamentID: itemReq.FilamentID,
				Quantity:   itemReq.Quantity,
				Order:      itemReq.Order,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if err := uc.budgetRepository.AddItem(ctx, item); err != nil {
				uc.logger.Error(ctx, "Failed to create budget item", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}

	// Save to repository
	if err := uc.budgetRepository.Update(ctx, budget); err != nil {
		uc.logger.Error(ctx, "Failed to update budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Recalculate costs
	if err := uc.budgetRepository.CalculateCosts(ctx, budget.ID); err != nil {
		uc.logger.Error(ctx, "Failed to recalculate budget costs", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Retrieve the updated budget
	budget, _ = uc.budgetRepository.FindByID(ctx, budget.ID, organizationID)

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

	uc.logger.Info(ctx, "Budget updated successfully", map[string]interface{}{
		"budget_id": budget.ID,
	})

	c.JSON(http.StatusOK, response)
}
