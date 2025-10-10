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

	// Create new budget as draft
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
		IncludeLaborCost:  originalBudget.IncludeLaborCost,
		IncludeWasteCost:  originalBudget.IncludeWasteCost,
		LaborCostPerHour:  originalBudget.LaborCostPerHour,
		OwnerUserID:       userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := uc.budgetRepository.Create(ctx, newBudget); err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Copy items
	originalItems, _ := uc.budgetRepository.GetItems(ctx, originalBudget.ID)
	for _, originalItem := range originalItems {
		newItem := &entities.BudgetItemEntity{
			ID:         uuid.New(),
			BudgetID:   newBudget.ID,
			FilamentID: originalItem.FilamentID,
			Quantity:   originalItem.Quantity,
			Order:      originalItem.Order,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		uc.budgetRepository.AddItem(ctx, newItem)
	}

	// Calculate costs
	uc.budgetRepository.CalculateCosts(ctx, newBudget.ID)

	// Return new budget
	newBudget, _ = uc.budgetRepository.FindByID(ctx, newBudget.ID, organizationID)
	customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, newBudget.CustomerID)
	items, _ := uc.budgetRepository.GetItems(ctx, newBudget.ID)

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
		Budget:   newBudget,
		Customer: customerInfo,
		Items:    itemResponses,
	}

	c.JSON(http.StatusCreated, response)
}

// Recalculate recalculates all costs for a budget
// @Summary Recalculate budget costs
// @Description Recalculate all costs for a budget
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
	_, err = uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Budget not found"})
		return
	}

	// Recalculate costs
	if err := uc.budgetRepository.CalculateCosts(ctx, budgetID); err != nil {
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Return updated budget
	budget, _ := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
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
		customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)
		items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)
		itemResponses := make([]entities.BudgetItemResponse, len(items))
		for j, item := range items {
			filamentInfo, _ := uc.budgetRepository.GetFilamentInfo(ctx, item.FilamentID)
			itemResponses[j] = entities.BudgetItemResponse{
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
		budgetResponses[i] = entities.BudgetResponse{
			Budget:   budget,
			Customer: customerInfo,
			Items:    itemResponses,
		}
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
