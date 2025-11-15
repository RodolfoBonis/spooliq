package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindByID retrieves a customer by ID
// @Summary Get customer by ID
// @Description Get a specific customer by ID
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} entities.CustomerResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers/{id} [get]
// @Security BearerAuth
func (uc *CustomerUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customer retrieval by ID attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		appError := coreErrors.UsecaseError("Organization ID not found in context")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Parse customer ID
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid customer ID", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid customer ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	customer, err := uc.repository.FindByID(ctx, customerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customer", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": customerID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	budgetCount, _ := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)

	budgets, err := uc.repository.GetCustomerBudgets(ctx, customer.ID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customer budgets", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": customer.ID,
		})
		budgets = []entities.BudgetSummary{}
	}

	response := entities.CustomerResponse{
		Customer:    customer,
		BudgetCount: int(budgetCount),
		Budgets:     budgets,
	}

	uc.logger.Info(ctx, "Customer retrieved successfully", map[string]interface{}{
		"customer_id": customer.ID,
	})

	c.JSON(http.StatusOK, response)
}
