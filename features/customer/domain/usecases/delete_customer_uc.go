package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Delete deletes a customer (soft delete)
// @Summary Delete customer
// @Description Delete a customer (soft delete)
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers/{id} [delete]
// @Security BearerAuth
func (uc *CustomerUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customer deletion attempt started", map[string]interface{}{
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

	// Check if customer exists and user has permission
	customer, err := uc.repository.FindByID(ctx, customerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customer", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": customerID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// Check if customer has associated budgets
	budgetCount, err := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check customer budgets", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": customerID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	if budgetCount > 0 {
		uc.logger.Error(ctx, "Cannot delete customer with associated budgets", map[string]interface{}{
			"customer_id":  customerID,
			"budget_count": budgetCount,
		})
		appError := coreErrors.UsecaseError("Customer has associated budgets and cannot be deleted")
		c.JSON(http.StatusConflict, gin.H{"error": appError.Message})
		return
	}

	// Delete customer
	if err := uc.repository.Delete(ctx, customerID); err != nil {
		uc.logger.Error(ctx, "Failed to delete customer", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Customer deleted successfully", map[string]interface{}{
		"customer_id": customerID,
	})

	c.Status(http.StatusNoContent)
}
