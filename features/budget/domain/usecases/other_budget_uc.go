package usecases

import (
	"context"
	"errors"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Duplicate duplicates an existing budget as a new draft
// @Summary Duplicate budget
// @Description Duplicate a budget as a new draft
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 201 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/duplicate [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Duplicate(c *gin.Context) {
	ctx := c.Request.Context()

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

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		appError := coreErrors.UsecaseError("Invalid budget ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get original budget
	originalBudget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	now := time.Now()

	// Create new budget as draft, copying ALL relevant fields (including the
	// PDF-facing ones that used to be dropped: delivery days, payment terms, notes).
	newBudget := &entities.BudgetEntity{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		Name:              originalBudget.Name + " (Copy)",
		Description:       originalBudget.Description,
		CustomerID:        originalBudget.CustomerID,
		Status:            entities.StatusDraft,
		PrintTimeHours:    originalBudget.PrintTimeHours,
		PrintTimeMinutes:  originalBudget.PrintTimeMinutes,
		MachinePresetID:   originalBudget.MachinePresetID,
		EnergyPresetID:    originalBudget.EnergyPresetID,
		CostPresetID:      originalBudget.CostPresetID,
		IncludeEnergyCost: originalBudget.IncludeEnergyCost,
		IncludeWasteCost:  originalBudget.IncludeWasteCost,
		DeliveryDays:      originalBudget.DeliveryDays,
		PaymentTerms:      originalBudget.PaymentTerms,
		Notes:             originalBudget.Notes,
		OwnerUserID:       userID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Read the original items + filaments and prepare fully-populated copies.
	originalItems, err := uc.budgetRepository.GetItems(ctx, originalBudget.ID)
	if err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	type itemBundle struct {
		item      *entities.BudgetItemEntity
		filaments []*entities.BudgetItemFilamentEntity
	}

	bundles := make([]itemBundle, 0, len(originalItems))
	for _, original := range originalItems {
		originalFilaments, err := uc.budgetRepository.GetItemFilaments(ctx, original.ID)
		if err != nil {
			appError := coreErrors.RepositoryError(err.Error())
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}

		newItemID := uuid.New()
		newItem := &entities.BudgetItemEntity{
			ID:                      newItemID,
			BudgetID:                newBudget.ID,
			FilamentID:              original.FilamentID,
			OrganizationID:          organizationID,
			Quantity:                original.Quantity,
			Order:                   original.Order,
			ProductName:             original.ProductName,
			ProductDescription:      original.ProductDescription,
			ProductQuantity:         original.ProductQuantity,
			ProductDimensions:       original.ProductDimensions,
			PrintTimeHours:          original.PrintTimeHours,
			PrintTimeMinutes:        original.PrintTimeMinutes,
			SetupTimeMinutes:        original.SetupTimeMinutes,
			ManualLaborMinutesTotal: original.ManualLaborMinutesTotal,
			CostPresetID:            original.CostPresetID,
			AdditionalNotes:         original.AdditionalNotes,
			CreatedAt:               now,
			UpdatedAt:               now,
		}

		newFilaments := make([]*entities.BudgetItemFilamentEntity, 0, len(originalFilaments))
		for _, original := range originalFilaments {
			newFilaments = append(newFilaments, &entities.BudgetItemFilamentEntity{
				ID:             uuid.New(),
				BudgetItemID:   newItemID,
				FilamentID:     original.FilamentID,
				OrganizationID: organizationID,
				Quantity:       original.Quantity,
				Order:          original.Order,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}

		bundles = append(bundles, itemBundle{item: newItem, filaments: newFilaments})
	}

	initialHistory := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       newBudget.ID,
		OrganizationID: organizationID,
		PreviousStatus: "",
		NewStatus:      entities.StatusDraft,
		ChangedBy:      userID,
		CreatedAt:      now,
	}

	// Persist the whole copy atomically.
	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.Create(ctx, newBudget); err != nil {
			return err
		}
		if err := repo.AddStatusHistory(ctx, initialHistory); err != nil {
			return err
		}
		for _, b := range bundles {
			if err := repo.AddItem(ctx, b.item); err != nil {
				return err
			}
			for _, filament := range b.filaments {
				if err := repo.AddItemFilament(ctx, filament); err != nil {
					return err
				}
			}
		}
		return repo.CalculateCosts(ctx, newBudget.ID, organizationID)
	}); err != nil {
		uc.logger.Error(ctx, "Failed to duplicate budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Return new budget
	response, err := uc.buildBudgetResponse(ctx, newBudget.ID, organizationID)
	if err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Recalculate recalculates all costs for a draft budget.
// @Summary Recalculate budget costs
// @Description Recalculate all costs for a budget. Only allowed while the budget is a draft.
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/recalculate [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Recalculate(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		appError := coreErrors.UsecaseError("Invalid budget ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Verify budget exists and user has permission
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Recalculation mutates stored costs, so it is only allowed for drafts.
	if budget.Status != entities.StatusDraft {
		uc.logger.Error(ctx, "Cannot recalculate non-draft budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		appError := coreErrors.ConflictError("Only draft budgets can be recalculated")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// A recalculation invalidates any previously generated PDF.
	budget.PDFUrl = nil
	budget.UpdatedAt = time.Now()

	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.Update(ctx, budget); err != nil {
			return err
		}
		return repo.CalculateCosts(ctx, budgetID, organizationID)
	}); err != nil {
		// A concurrent status change (e.g. approval) makes the draft no longer
		// recalculable; surface that as a conflict rather than a 500.
		if errors.Is(err, entities.ErrBudgetNotEditable) {
			appError := coreErrors.ConflictError("Only draft budgets can be recalculated")
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Return updated budget
	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}
	c.JSON(http.StatusOK, response)
}

// GetCalculation returns the currently stored costs for a budget without
// recalculating. It is read-only and safe to call on budgets in any status.
//
// Deprecated: use POST /v1/budgets/{id}/recalculate to recompute costs. This GET
// endpoint previously recalculated on read, which mutated already-approved
// budgets; it now only returns the stored values and is kept for backward
// compatibility.
// @Summary Get budget costs (deprecated)
// @Description Deprecated: returns stored budget costs without recalculating. Use POST /v1/budgets/{id}/recalculate to recompute.
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/calculate [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GetCalculation(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		appError := coreErrors.UsecaseError("Invalid budget ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// FindByCustomer retrieves all budgets for a specific customer
// @Summary List budgets by customer
// @Description Get all budgets for a specific customer
// @Tags budgets
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} entities.ListBudgetsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/by-customer/{customer_id} [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindByCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		appError := coreErrors.UsecaseError("Invalid customer ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get budgets
	budgets, err := uc.budgetRepository.FindByCustomer(ctx, customerID, organizationID)
	if err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response
	budgetResponses := make([]entities.BudgetResponse, len(budgets))
	for i, budget := range budgets {
		response, _ := uc.buildBudgetResponse(ctx, budget.ID, organizationID)
		budgetResponses[i] = *response
	}

	total := len(budgets)
	response := entities.ListBudgetsResponse{
		Data:       budgetResponses,
		Total:      total,
		Page:       1,
		PageSize:   total,
		TotalPages: 1,
	}

	c.JSON(http.StatusOK, response)
}

// GetHistory retrieves the status history for a budget
// @Summary Get budget status history
// @Description Get the status change history for a budget
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} []entities.BudgetStatusHistoryEntity
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/history [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		appError := coreErrors.UsecaseError("Invalid budget ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Verify budget exists and user has permission
	_, err = uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Get history
	history, err := uc.budgetRepository.GetStatusHistory(ctx, budgetID)
	if err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, history)
}

// buildBudgetResponse builds a complete budget response with items and filaments.
// It delegates to the shared package-level builder (response_builder.go) so the
// sale distribution and cost-preset resolution live in exactly one place.
func (uc *BudgetUseCase) buildBudgetResponse(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetResponse, error) {
	return buildBudgetResponse(ctx, uc.budgetRepository, budgetID, organizationID)
}
