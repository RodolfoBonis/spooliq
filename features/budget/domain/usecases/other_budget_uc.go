package usecases

import (
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
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/duplicate [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Duplicate(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeUserRequired, "Usuário não identificado"))
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	// Get original budget
	originalBudget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
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
		ProfileID:         originalBudget.ProfileID,
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
		respondBudgetError(c, err)
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
			respondBudgetError(c, err)
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
			Model3DID:               original.Model3DID,
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
		uc.logger.Error(ctx, "Failed to duplicate budget", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	// Return new budget
	response, err := uc.buildBudgetResponse(ctx, newBudget.ID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
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
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/recalculate [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Recalculate(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	// Verify budget exists and user has permission
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	// Recalculation mutates stored costs, so it is only allowed for drafts.
	if budget.Status != entities.StatusDraft {
		uc.logger.Error(ctx, "Cannot recalculate non-draft budget", map[string]interface{}{
			"budget_id": budgetID,
			"status":    budget.Status,
		})
		coreErrors.Respond(c, coreErrors.Conflict(CodeBudgetNotEditable, "Apenas orçamentos em rascunho podem ser recalculados"))
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
			coreErrors.Respond(c, coreErrors.Conflict(CodeBudgetNotEditable, "Apenas orçamentos em rascunho podem ser recalculados"))
			return
		}
		uc.logger.Error(ctx, "Failed to recalculate budget", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	// Return updated budget
	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
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
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/calculate [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GetCalculation(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// FindByCustomer retrieves budgets for a specific customer (paginated).
// @Summary List budgets by customer
// @Description List budgets for a specific customer (paginated). Accepts the same
// @Description pagination, sort and filter params as GET /budgets (status, q,
// @Description from/to); the path customer_id always scopes the result.
// @Tags budgets
// @Accept json
// @Produce json
// @Param customer_id path string true "Customer ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size (max 100)" default(20)
// @Param q query string false "Free-text search on budget name (case-insensitive)"
// @Param status query string false "Filter by status" Enums(draft, sent, approved, rejected, printing, completed)
// @Param from query string false "Created-at lower bound (YYYY-MM-DD or RFC3339, inclusive)"
// @Param to query string false "Created-at upper bound (YYYY-MM-DD or RFC3339, inclusive)"
// @Param sort_by query string false "Sort field" Enums(created_at, name, total_cost, status) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} helpers.Page[entities.BudgetResponse]
// @Failure 400 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/by-customer/{customer_id} [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindByCustomer(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	customerID, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidCustomerID, "ID de cliente inválido"))
		return
	}

	listQuery := budgetListQuery(c)

	filters, apiErr := parseBudgetFilters(c, listQuery.Search)
	if apiErr != nil {
		coreErrors.Respond(c, apiErr)
		return
	}
	// The path customer_id always scopes the result (overriding any query param).
	filters["customer_id"] = customerID

	budgets, total, err := uc.budgetRepository.SearchBudgets(ctx, organizationID, filters, listQuery.OrderClause(), listQuery.Limit(), listQuery.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budgets by customer", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	budgetResponses, err := uc.buildBudgetListResponses(ctx, budgets, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to build budget list response", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(budgetResponses, int64(total), listQuery))
}

// GetHistory retrieves the status history for a budget (paginated).
// @Summary Get budget status history
// @Description Get the status change history for a budget (paginated, newest first).
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size (max 100)" default(50)
// @Success 200 {object} helpers.Page[entities.BudgetStatusHistoryEntity]
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/history [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GetHistory(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	// Verify budget exists within the organization.
	if _, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID); err != nil {
		respondBudgetError(c, err)
		return
	}

	listQuery := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: budgetHistoryPageSize})

	history, total, err := uc.budgetRepository.GetStatusHistory(ctx, budgetID, listQuery.Limit(), listQuery.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to get budget status history", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(history, total, listQuery))
}
