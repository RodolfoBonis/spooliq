package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID retrieves a customer by ID.
// @Summary Get customer by ID
// @Description Get a specific customer by ID
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} entities.CustomerResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers/{id} [get]
// @Security BearerAuth
func (uc *CustomerUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid customer ID", map[string]interface{}{"customer_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_customer_id", "ID de cliente inválido"))
		return
	}

	customer, err := uc.repository.FindByID(ctx, customerID, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Customer not found", map[string]interface{}{"customer_id": customerID})
			coreErrors.Respond(c, coreErrors.NotFoundErr("customer_not_found", "Cliente não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve customer", map[string]interface{}{"customer_id": customerID, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	budgetCount, _ := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)
	totalBudgets, _ := uc.repository.SumBudgetTotalsByCustomerAndStatus(ctx, customer.ID, budgetTotalStatuses)

	budgets, err := uc.repository.GetCustomerBudgets(ctx, customer.ID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customer budgets", map[string]interface{}{"customer_id": customer.ID, "error": err.Error()})
		budgets = []entities.BudgetSummary{}
	}

	response := entities.CustomerResponse{
		Customer:     customer,
		BudgetCount:  int(budgetCount),
		TotalBudgets: &totalBudgets,
		Budgets:      budgets,
	}

	uc.logger.Info(ctx, "Customer retrieved successfully", map[string]interface{}{"customer_id": customer.ID})

	c.JSON(http.StatusOK, response)
}
