package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
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
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var request entities.CreateBudgetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind request", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(&request); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Validate that each item has at least one filament
	for i, item := range request.Items {
		if len(item.Filaments) == 0 {
			uc.logger.Error(ctx, "Item has no filaments", map[string]interface{}{"item_index": i})
			coreErrors.Respond(c, coreErrors.BadRequest(coreErrors.CodeValidationError, "Cada item do orçamento deve ter ao menos um filamento"))
			return
		}
	}

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

	// Check if customer exists and user has permission
	if _, err := uc.customerRepository.FindByID(ctx, request.CustomerID, organizationID); err != nil {
		uc.logger.Error(ctx, "Customer not found", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": request.CustomerID,
		})
		coreErrors.Respond(c, coreErrors.NotFoundErr(CodeCustomerNotFound, "Cliente não encontrado"))
		return
	}

	// Resolve the machine/energy/cost presets from the request, the requested
	// profile, the org's default profile and the org's default presets (in that
	// precedence). The resolver validates the profile and every resolved preset ID
	// against the organization, so a bad reference surfaces as a 400 here.
	resolved, err := uc.resolvePresets(c, organizationID, PresetResolutionInput{
		ProfileID:       request.ProfileID,
		MachinePresetID: request.MachinePresetID,
		EnergyPresetID:  request.EnergyPresetID,
		CostPresetID:    request.CostPresetID,
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to resolve budget presets", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	// Validate the remaining references (item-level cost presets + filaments). The
	// budget-level machine/energy/cost presets were already validated by the resolver.
	if err := uc.validateReferences(ctx, organizationID, nil, nil, nil, request.Items); err != nil {
		uc.logger.Error(ctx, "Invalid budget references", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
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
		ProfileID:         resolved.ProfileID,
		MachinePresetID:   resolved.MachinePresetID,
		EnergyPresetID:    resolved.EnergyPresetID,
		CostPresetID:      resolved.CostPresetID,
		IncludeEnergyCost: request.IncludeEnergyCost,
		IncludeWasteCost:  request.IncludeWasteCost,
		DeliveryDays:      request.DeliveryDays,
		PaymentTerms:      request.PaymentTerms,
		Notes:             request.Notes,
		OwnerUserID:       userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Record initial status history for the draft status
	initialHistory := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       budget.ID,
		OrganizationID: organizationID,
		PreviousStatus: "",
		NewStatus:      entities.StatusDraft,
		ChangedBy:      userID,
		CreatedAt:      time.Now(),
	}

	// Build items (+ filaments) consistently with OrganizationID and legacy columns.
	built := buildBudgetItems(budget.ID, organizationID, request.Items)

	// Persist everything atomically: budget, status history, items, filaments and
	// cost calculation all succeed together or roll back together.
	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.Create(ctx, budget); err != nil {
			return err
		}
		if err := repo.AddStatusHistory(ctx, initialHistory); err != nil {
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
		return repo.CalculateCosts(ctx, budget.ID, organizationID)
	}); err != nil {
		uc.logger.Error(ctx, "Failed to create budget", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	// Build the response from the freshly stored (and costed) budget.
	response, err := uc.buildBudgetResponse(ctx, budget.ID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve created budget", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	uc.logger.Info(ctx, "Budget created successfully", map[string]interface{}{
		"budget_id": budget.ID,
		"name":      budget.Name,
	})

	c.JSON(http.StatusCreated, response)

	// Record activity (fire-and-forget)
	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budget.ID.String(),
		EntityName:     budget.Name,
		CreatedAt:      time.Now(),
	})
}
