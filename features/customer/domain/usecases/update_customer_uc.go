package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Update updates an existing customer
// @Summary Update customer
// @Description Update an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param request body entities.UpdateCustomerRequest true "Update customer request"
// @Success 200 {object} entities.CustomerResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers/{id} [put]
// @Security BearerAuth
func (uc *CustomerUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customer update attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	userID := getUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		appError := coreErrors.UsecaseError("User ID not found in context")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	admin := isAdmin(c)

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

	var request entities.UpdateCustomerRequest
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

	// Get existing customer
	customer, err := uc.repository.FindByID(ctx, customerID, userID, admin)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customer", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": customerID,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// Check if email is being changed and if it already exists
	if request.Email != nil && *request.Email != "" {
		if customer.Email == nil || *customer.Email != *request.Email {
			exists, err := uc.repository.ExistsByEmail(ctx, *request.Email, userID, &customerID)
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
	}

	// Update fields
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

	customer.UpdatedAt = time.Now()

	// Save to repository
	if err := uc.repository.Update(ctx, customer); err != nil {
		uc.logger.Error(ctx, "Failed to update customer", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Customer updated successfully", map[string]interface{}{
		"customer_id": customer.ID,
	})

	// Get budget count
	budgetCount, _ := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)

	response := entities.CustomerResponse{
		Customer:    customer,
		BudgetCount: int(budgetCount),
	}

	c.JSON(http.StatusOK, response)
}
