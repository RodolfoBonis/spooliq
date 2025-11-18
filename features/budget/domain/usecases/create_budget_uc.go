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

// Create creates a new budget
// @Summary Create budget
// @Description Create a new budget with items
// @Tags budgets
// @Accept json
// @Produce json
// @Param request body entities.CreateBudgetRequest true "Create budget request"
// @Success 201 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Budget creation attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	var request entities.CreateBudgetRequest
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

	// Validate that each item has at least one filament
	for i, item := range request.Items {
		if len(item.Filaments) == 0 {
			uc.logger.Error(ctx, "Item has no filaments", map[string]interface{}{
				"item_index": i,
			})
			appError := coreErrors.UsecaseError("Each item must have at least one filament")
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
	}

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

	// Check if customer exists and user has permission
	_, err := uc.customerRepository.FindByID(ctx, request.CustomerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Customer not found", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": request.CustomerID,
		})
		appError := coreErrors.UsecaseError("Customer not found")
		c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
		return
	}

	// Create budget entity (without global print time - now calculated from items)
	budget := &entities.BudgetEntity{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		Name:              request.Name,
		Description:       request.Description,
		CustomerID:        request.CustomerID,
		Status:            entities.StatusDraft,
		MachinePresetID:   request.MachinePresetID,
		EnergyPresetID:    request.EnergyPresetID,
		IncludeEnergyCost: request.IncludeEnergyCost,
		IncludeWasteCost:  request.IncludeWasteCost,
		DeliveryDays:      request.DeliveryDays,
		PaymentTerms:      request.PaymentTerms,
		Notes:             request.Notes,
		OwnerUserID:       userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Save budget
	if err := uc.budgetRepository.Create(ctx, budget); err != nil {
		uc.logger.Error(ctx, "Failed to create budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Create budget items (products) with their filaments
	for _, itemReq := range request.Items {
		// Use the first filament as the primary filament (for backward compatibility)
		var primaryFilamentID uuid.UUID
		if len(itemReq.Filaments) > 0 {
			primaryFilamentID = itemReq.Filaments[0].FilamentID
		}

		item := &entities.BudgetItemEntity{
			ID:                      uuid.New(),
			BudgetID:                budget.ID,
			FilamentID:              primaryFilamentID,
			OrganizationID:          organizationID,
			ProductName:             itemReq.ProductName,
			ProductDescription:      itemReq.ProductDescription,
			ProductQuantity:         itemReq.ProductQuantity,
			ProductDimensions:       itemReq.ProductDimensions,
			PrintTimeHours:          itemReq.PrintTimeHours,
			PrintTimeMinutes:        itemReq.PrintTimeMinutes,
			SetupTimeMinutes:        itemReq.SetupTimeMinutes,
			ManualLaborMinutesTotal: itemReq.ManualLaborMinutesTotal,
			CostPresetID:            itemReq.CostPresetID,
			AdditionalNotes:         itemReq.AdditionalNotes,
			Order:                   itemReq.Order,
			CreatedAt:               time.Now(),
			UpdatedAt:               time.Now(),
		}

		// Save item
		if err := uc.budgetRepository.AddItem(ctx, item); err != nil {
			uc.logger.Error(ctx, "Failed to create budget item", map[string]interface{}{
				"error": err.Error(),
			})
			// Rollback: delete the budget
			uc.budgetRepository.Delete(ctx, budget.ID)
			appError := coreErrors.RepositoryError(err.Error())
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}

		// Create filaments for this item
		for _, filReq := range itemReq.Filaments {
			filament := &entities.BudgetItemFilamentEntity{
				ID:             uuid.New(),
				BudgetItemID:   item.ID,
				FilamentID:     filReq.FilamentID,
				OrganizationID: organizationID,
				Quantity:       filReq.Quantity,
				Order:          filReq.Order,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			if err := uc.budgetRepository.AddItemFilament(ctx, filament); err != nil {
				uc.logger.Error(ctx, "Failed to add filament to item", map[string]interface{}{
					"error":       err.Error(),
					"item_id":     item.ID,
					"filament_id": filReq.FilamentID,
				})
				// Rollback: delete the budget
				uc.budgetRepository.Delete(ctx, budget.ID)
				appError := coreErrors.RepositoryError("Failed to add filament: " + err.Error())
				c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
				return
			}
		}
	}

	// Calculate costs
	if err := uc.budgetRepository.CalculateCosts(ctx, budget.ID); err != nil {
		uc.logger.Error(ctx, "Failed to calculate budget costs", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Retrieve the updated budget with calculated costs
	budget, err = uc.budgetRepository.FindByID(ctx, budget.ID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve created budget", map[string]interface{}{
			"error": err.Error(),
		})
	}

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
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			CostPresetID:            costPresetIDStr,
			AdditionalNotes:         item.AdditionalNotes,
			FilamentCost:            item.FilamentCost,
			WasteCost:               item.WasteCost,
			EnergyCost:              item.EnergyCost,
			SetupCost:               item.SetupCost,
			ManualLaborCost:         item.ManualLaborCost,
			ItemTotalCost:           item.ItemTotalCost,
			UnitPrice:               item.UnitPrice,
			Filaments:               filaments,
			Order:                   item.Order,
			CreatedAt:               item.CreatedAt,
			UpdatedAt:               item.UpdatedAt,
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

	uc.logger.Info(ctx, "Budget created successfully", map[string]interface{}{
		"budget_id": budget.ID,
		"name":      budget.Name,
	})

	c.JSON(http.StatusCreated, response)
}
