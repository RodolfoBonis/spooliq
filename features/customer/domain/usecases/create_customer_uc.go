package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create creates a new customer.
// @Summary Create customer
// @Description Create a new customer
// @Tags customers
// @Accept json
// @Produce json
// @Param request body entities.CreateCustomerRequest true "Create customer request"
// @Success 201 {object} entities.CustomerResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers [post]
// @Security BearerAuth
func (uc *CustomerUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("user_required", "Usuário não encontrado no contexto"))
		return
	}

	var request entities.CreateCustomerRequest
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

	if request.Email != nil && *request.Email != "" {
		exists, err := uc.repository.ExistsByEmail(ctx, *request.Email, organizationID, nil)
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

	customer := &entities.CustomerEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           request.Name,
		Email:          request.Email,
		Phone:          request.Phone,
		Document:       request.Document,
		Address:        request.Address,
		City:           request.City,
		State:          request.State,
		ZipCode:        request.ZipCode,
		Notes:          request.Notes,
		OwnerUserID:    userID,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := uc.repository.Create(ctx, customer); err != nil {
		uc.logger.Error(ctx, "Failed to create customer", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Customer created successfully", map[string]interface{}{"customer_id": customer.ID})

	c.JSON(http.StatusCreated, entities.CustomerResponse{Customer: customer, BudgetCount: 0})

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityCustomer,
		EntityID:       customer.ID.String(),
		EntityName:     customer.Name,
		Description:    "Customer created: " + customer.Name,
	})
}
