package usecases

import (
	"fmt"
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
	if request.MachinePresetID != nil {
		budget.MachinePresetID = request.MachinePresetID
	}
	if request.EnergyPresetID != nil {
		budget.EnergyPresetID = request.EnergyPresetID
	}
	if request.IncludeEnergyCost != nil {
		budget.IncludeEnergyCost = *request.IncludeEnergyCost
	}
	if request.IncludeWasteCost != nil {
		budget.IncludeWasteCost = *request.IncludeWasteCost
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

	// Update items if provided (delete all and recreate)
	if request.Items != nil {
		// Validate items
		for i, item := range *request.Items {
			if len(item.Filaments) == 0 {
				uc.logger.Error(ctx, "Item has no filaments", map[string]interface{}{
					"item_index": i,
				})
				appError := coreErrors.UsecaseError("Each item must have at least one filament")
				c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
				return
			}
		}

		// Delete all existing items (cascade will delete filaments)
		if err := uc.budgetRepository.DeleteAllItems(ctx, budgetID); err != nil {
			uc.logger.Error(ctx, "Failed to delete existing items", map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Create new items with filaments
		for _, itemReq := range *request.Items {
			item := &entities.BudgetItemEntity{
				ID:                  uuid.New(),
				BudgetID:            budget.ID,
				ProductName:         itemReq.ProductName,
				ProductDescription:  itemReq.ProductDescription,
				ProductQuantity:     itemReq.ProductQuantity,
				ProductDimensions:   itemReq.ProductDimensions,
				PrintTimeHours:      itemReq.PrintTimeHours,
				PrintTimeMinutes:    itemReq.PrintTimeMinutes,
				CostPresetID:        itemReq.CostPresetID,
				SetupTimeMinutes:        itemReq.SetupTimeMinutes,
			ManualLaborMinutesTotal: itemReq.ManualLaborMinutesTotal,
				AdditionalNotes:     itemReq.AdditionalNotes,
				Order:               itemReq.Order,
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
			}

			// Save item
			if err := uc.budgetRepository.AddItem(ctx, item); err != nil {
				uc.logger.Error(ctx, "Failed to create budget item", map[string]interface{}{
					"error": err.Error(),
				})
				continue
			}

			// Create filaments for this item
			for _, filReq := range itemReq.Filaments {
				filament := &entities.BudgetItemFilamentEntity{
					ID:           uuid.New(),
					BudgetItemID: item.ID,
					FilamentID:   filReq.FilamentID,
					Quantity:     filReq.Quantity,
					Order:        filReq.Order,
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}

				if err := uc.budgetRepository.AddItemFilament(ctx, filament); err != nil {
					uc.logger.Error(ctx, "Failed to add filament to item", map[string]interface{}{
						"error":       err.Error(),
						"item_id":     item.ID,
						"filament_id": filReq.FilamentID,
					})
				}
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

	response := entities.BudgetResponse{
		BudgetEntity:          budget,
		Customer:              customerInfo,
		Items:                 itemResponses,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: totalPrintTimeDisplay,
	}

	uc.logger.Info(ctx, "Budget updated successfully", map[string]interface{}{
		"budget_id": budget.ID,
	})

	c.JSON(http.StatusOK, response)
}
