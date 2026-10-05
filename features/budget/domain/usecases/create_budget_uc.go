package usecases

import (
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

	// Validate that every referenced preset/filament belongs to the organization.
	if err := uc.validateReferences(ctx, organizationID, request.MachinePresetID, request.EnergyPresetID, nil, request.Items); err != nil {
		uc.logger.Error(ctx, "Invalid budget references", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
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
		uc.logger.Error(ctx, "Failed to create budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build the response from the freshly stored (and costed) budget.
	response, err := buildBudgetResponse(ctx, uc.budgetRepository, budget.ID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve created budget", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
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
