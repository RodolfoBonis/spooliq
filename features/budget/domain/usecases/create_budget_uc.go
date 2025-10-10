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

	// Validate print time
	if request.PrintTimeHours == 0 && request.PrintTimeMinutes == 0 {
		uc.logger.Error(ctx, "Invalid print time", nil)
		appError := coreErrors.UsecaseError("Print time must be greater than zero")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
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

	// Create budget entity
	budget := &entities.BudgetEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           request.Name,
		Description:    request.Description,
		CustomerID:     request.CustomerID,
		Status:         entities.StatusDraft,
		PrintTimeHours:    request.PrintTimeHours,
		PrintTimeMinutes:  request.PrintTimeMinutes,
		MachinePresetID:   request.MachinePresetID,
		EnergyPresetID:    request.EnergyPresetID,
		CostPresetID:      request.CostPresetID,
		IncludeEnergyCost: request.IncludeEnergyCost,
		IncludeLaborCost:  request.IncludeLaborCost,
		IncludeWasteCost:  request.IncludeWasteCost,
		LaborCostPerHour:  request.LaborCostPerHour,
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

	// Create budget items
	for _, itemReq := range request.Items {
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
			// Rollback: delete the budget
			uc.budgetRepository.Delete(ctx, budget.ID)
			appError := coreErrors.RepositoryError(err.Error())
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
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

	uc.logger.Info(ctx, "Budget created successfully", map[string]interface{}{
		"budget_id": budget.ID,
		"name":      budget.Name,
	})

	c.JSON(http.StatusCreated, response)
}
