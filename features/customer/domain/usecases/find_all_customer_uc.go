package usecases

import (
	"net/http"
	"strconv"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
)

// FindAll retrieves all customers with pagination
// @Summary List customers
// @Description Get all customers with pagination
// @Tags customers
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} entities.ListCustomersResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers [get]
// @Security BearerAuth
func (uc *CustomerUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customers retrieval attempt started", map[string]interface{}{
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

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	// Get customers from repository
	customers, total, err := uc.repository.FindAll(ctx, organizationID, pageSize, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customers", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build response
	customerResponses := make([]entities.CustomerResponse, len(customers))
	for i, customer := range customers {
		// Get budget count for each customer
		budgetCount, _ := uc.repository.CountBudgetsByCustomer(ctx, customer.ID)
		customerResponses[i] = entities.CustomerResponse{
			Customer:    customer,
			BudgetCount: int(budgetCount),
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := entities.ListCustomersResponse{
		Data:       customerResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	uc.logger.Info(ctx, "Customers retrieved successfully", map[string]interface{}{
		"count": len(customers),
		"total": total,
		"page":  page,
	})

	c.JSON(http.StatusOK, response)
}
