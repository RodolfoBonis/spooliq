package usecases

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
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
// @Failure 409 {object} map[string]interface{}
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

	// Check if budget can be edited. This is a fast pre-check; the authoritative
	// guard is the draft-scoped WHERE in the repository Update inside the
	// transaction, which closes the TOCTOU race with a concurrent status change.
	if !budget.CanBeEdited() {
		uc.logger.Error(ctx, "Cannot edit non-draft budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		appError := coreErrors.ConflictError("Only draft budgets can be edited")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Update fields (only those explicitly provided in the request)
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

	// A stored PDF no longer reflects the budget once it is edited.
	budget.PDFUrl = nil

	// Validate ONLY the references explicitly provided in this request. References
	// already stored on the budget are deliberately NOT re-validated: the preset
	// they point to may have been soft-deleted since, and historical references
	// remain usable for calculation (see repository CalculateCosts/GetPresetInfo).
	var itemsForValidation []entities.BudgetItemRequest
	if request.Items != nil {
		itemsForValidation = *request.Items
	}
	if err := uc.validateReferences(ctx, organizationID, request.MachinePresetID, request.EnergyPresetID, nil, itemsForValidation); err != nil {
		uc.logger.Error(ctx, "Invalid budget references", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// If items are provided, validate each has at least one filament before any write.
	var built []builtBudgetItem
	if request.Items != nil {
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
		built = buildBudgetItems(budget.ID, organizationID, *request.Items)
	}

	// Persist atomically: the draft-guarded budget update runs first so a concurrent
	// status change aborts the whole transaction before any item is touched; item
	// replacement (if any) and the cost recalculation then succeed or roll back
	// together.
	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.Update(ctx, budget); err != nil {
			return err
		}

		if request.Items != nil {
			if err := repo.DeleteAllItems(ctx, budgetID, organizationID); err != nil {
				return err
			}
			for _, b := range built {
				if err := repo.AddItem(ctx, b.Item); err != nil {
					return err
				}
				for _, filament := range b.Filaments {
					if err := repo.AddItemFilament(ctx, filament); err != nil {
						return err
					}
				}
			}
		}

		return repo.CalculateCosts(ctx, budget.ID)
	}); err != nil {
		// A concurrent approval (or deletion) turns this into a conflict rather
		// than an internal error.
		if errors.Is(err, entities.ErrBudgetNotEditable) {
			uc.logger.Warning(ctx, "Budget no longer editable (concurrent change)", map[string]interface{}{
				"budget_id": budgetID,
			})
			appError := coreErrors.ConflictError("Only draft budgets can be edited")
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		uc.logger.Error(ctx, "Failed to update budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Retrieve the updated budget
	budget, err = uc.budgetRepository.FindByID(ctx, budget.ID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve updated budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response
	customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
	items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)

	itemResponses := make([]entities.BudgetItemResponse, len(items))
	var totalPrintMinutes int

	for i, item := range items {
		// Get filament usage info for this item
		filaments, _ := uc.budgetRepository.GetFilamentUsageInfo(ctx, item.ID, organizationID)

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

	uc.logger.Info(ctx, "Budget updated successfully", map[string]interface{}{
		"budget_id": budget.ID,
	})

	c.JSON(http.StatusOK, response)

	// Record activity (fire-and-forget)
	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budget.ID.String(),
		EntityName:     budget.Name,
		CreatedAt:      time.Now(),
	})
}
