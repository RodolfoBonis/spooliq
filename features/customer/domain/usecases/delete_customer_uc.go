package usecases

import (
	"errors"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/database"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete deletes a customer (soft delete).
// @Summary Delete customer
// @Description Delete a customer (soft delete). Blocked when the customer has budgets.
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 204
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers/{id} [delete]
// @Security BearerAuth
func (uc *CustomerUseCase) Delete(c *gin.Context) {
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

	budgetCount, err := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check customer budgets", map[string]interface{}{"customer_id": customerID, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if budgetCount > 0 {
		uc.logger.Warning(ctx, "Cannot delete customer with associated budgets", map[string]interface{}{"customer_id": customerID, "budget_count": budgetCount})
		coreErrors.Respond(c, coreErrors.Conflict("customer_in_use", "Cliente possui orçamentos associados e não pode ser removido"))
		return
	}

	if err := uc.repository.Delete(ctx, customerID); err != nil {
		if database.IsForeignKeyViolation(err) {
			uc.logger.Warning(ctx, "Customer deletion blocked: customer in use", map[string]interface{}{"customer_id": customerID})
			coreErrors.Respond(c, coreErrors.Conflict("customer_in_use", "Cliente possui orçamentos associados e não pode ser removido"))
			return
		}
		uc.logger.Error(ctx, "Failed to delete customer", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Customer deleted successfully", map[string]interface{}{"customer_id": customerID})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityCustomer,
		EntityID:       customer.ID.String(),
		EntityName:     customer.Name,
		Description:    "Customer deleted: " + customer.Name,
	})
}
