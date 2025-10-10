package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create creates a new customer
// @Summary Create customer
// @Description Create a new customer
// @Tags customers
// @Accept json
// @Produce json
// @Param request body entities.CreateCustomerRequest true "Create customer request"
// @Success 201 {object} entities.CustomerResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers [post]
// @Security BearerAuth
func (uc *CustomerUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customer creation attempt started", map[string]interface{}{
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

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		appError := coreErrors.UsecaseError("User ID not found in context")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	var request entities.CreateCustomerRequest
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

	// Check if email already exists for this organization
	if request.Email != nil && *request.Email != "" {
		exists, err := uc.repository.ExistsByEmail(ctx, *request.Email, organizationID, nil)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check email existence", map[string]interface{}{
				"error": err.Error(),
			})
			appError := coreErrors.RepositoryError(err.Error())
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}

		if exists {
			uc.logger.Error(ctx, "Customer with email already exists", map[string]interface{}{
				"email": *request.Email,
			})
			appError := coreErrors.UsecaseError("Customer with this email already exists")
			c.JSON(http.StatusConflict, gin.H{"error": appError.Message})
			return
		}
	}

	// Create customer entity
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

	// Save to repository
	if err := uc.repository.Create(ctx, customer); err != nil {
		uc.logger.Error(ctx, "Failed to create customer", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Customer created successfully", map[string]interface{}{
		"customer_id": customer.ID,
		"name":        customer.Name,
	})

	response := entities.CustomerResponse{
		Customer:    customer,
		BudgetCount: 0,
	}

	c.JSON(http.StatusCreated, response)
}
