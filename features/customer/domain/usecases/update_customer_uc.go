package usecases

import (
	"errors"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update updates an existing customer.
// @Summary Update customer
// @Description Update an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param request body entities.UpdateCustomerRequest true "Update customer request"
// @Success 200 {object} entities.CustomerResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers/{id} [put]
// @Security BearerAuth
func (uc *CustomerUseCase) Update(c *gin.Context) {
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

	var request entities.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind request", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
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

	// Enforce unique email within the organization when it changes.
	if request.Email != nil && *request.Email != "" {
		if customer.Email == nil || *customer.Email != *request.Email {
			exists, err := uc.repository.ExistsByEmail(ctx, *request.Email, organizationID, &customerID)
			if err != nil {
				uc.logger.Error(ctx, "Failed to check email existence", map[string]interface{}{"error": err.Error()})
				coreErrors.Respond(c, err)
				return
			}
			if exists {
				uc.logger.Warning(ctx, "Customer with email already exists", map[string]interface{}{"email": *request.Email})
				coreErrors.Respond(c, coreErrors.Conflict("customer_email_taken", "Já existe um cliente com este e-mail"))
				return
			}
		}
	}

	applyCustomerUpdate(customer, &request)
	customer.UpdatedAt = time.Now()

	if err := uc.repository.Update(ctx, customer); err != nil {
		uc.logger.Error(ctx, "Failed to update customer", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Customer updated successfully", map[string]interface{}{"customer_id": customer.ID})

	budgetCount, _ := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)

	c.JSON(http.StatusOK, entities.CustomerResponse{Customer: customer, BudgetCount: int(budgetCount)})

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityCustomer,
		EntityID:       customer.ID.String(),
		EntityName:     customer.Name,
		Description:    "Customer updated: " + customer.Name,
	})
}

// applyCustomerUpdate copies the non-nil fields of the request onto the entity.
func applyCustomerUpdate(customer *entities.CustomerEntity, request *entities.UpdateCustomerRequest) {
	if request.Name != nil {
		customer.Name = *request.Name
	}
	if request.Email != nil {
		customer.Email = request.Email
	}
	if request.Phone != nil {
		customer.Phone = request.Phone
	}
	if request.Document != nil {
		customer.Document = request.Document
	}
	if request.Address != nil {
		customer.Address = request.Address
	}
	if request.City != nil {
		customer.City = request.City
	}
	if request.State != nil {
		customer.State = request.State
	}
	if request.ZipCode != nil {
		customer.ZipCode = request.ZipCode
	}
	if request.Notes != nil {
		customer.Notes = request.Notes
	}
	if request.IsActive != nil {
		customer.IsActive = *request.IsActive
	}
}
