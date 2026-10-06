package usecases

import (
	"errors"
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
// @Router /budgets/{id} [put]
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
	// Preset resolution on update is conditional: only when the request explicitly
	// sends profile_id do we re-resolve every slot from the profile/org defaults
	// (explicit IDs in the same request still win per slot). Otherwise we keep the
	// stored presets and apply only the explicit per-slot overrides, preserving the
	// existing partial-update semantics. presetsValidatedByResolver tracks whether
	// the resolver already validated the budget-level presets.
	presetsValidatedByResolver := false
	if request.ProfileID != nil {
		resolved, rerr := uc.resolvePresets(c, organizationID, PresetResolutionInput{
			ProfileID:       request.ProfileID,
			MachinePresetID: request.MachinePresetID,
			EnergyPresetID:  request.EnergyPresetID,
			CostPresetID:    request.CostPresetID,
		})
		if rerr != nil {
			uc.logger.Error(ctx, "Failed to resolve budget presets", map[string]interface{}{
				"error": rerr.Error(),
			})
			appError := coreErrors.BadRequestError(rerr.Error())
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		budget.ProfileID = resolved.ProfileID
		budget.MachinePresetID = resolved.MachinePresetID
		budget.EnergyPresetID = resolved.EnergyPresetID
		budget.CostPresetID = resolved.CostPresetID
		presetsValidatedByResolver = true
	} else {
		if request.MachinePresetID != nil {
			budget.MachinePresetID = request.MachinePresetID
		}
		if request.EnergyPresetID != nil {
			budget.EnergyPresetID = request.EnergyPresetID
		}
		if request.CostPresetID != nil {
			budget.CostPresetID = request.CostPresetID
		}
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
	var vMachine, vEnergy, vCost *uuid.UUID
	if !presetsValidatedByResolver {
		vMachine, vEnergy, vCost = request.MachinePresetID, request.EnergyPresetID, request.CostPresetID
	}
	if err := uc.validateReferences(ctx, organizationID, vMachine, vEnergy, vCost, itemsForValidation); err != nil {
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

		return repo.CalculateCosts(ctx, budget.ID, organizationID)
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

	// Build the response from the freshly stored (and recosted) budget.
	response, err := uc.buildBudgetResponse(ctx, budget.ID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve updated budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
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
