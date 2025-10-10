package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
)

// Search searches for customers with filters
// @Summary Search customers
// @Description Search customers with various filters
// @Tags customers
// @Accept json
// @Produce json
// @Param name query string false "Customer name"
// @Param email query string false "Customer email"
// @Param phone query string false "Customer phone"
// @Param document query string false "Customer document (CPF/CNPJ)"
// @Param city query string false "Customer city"
// @Param state query string false "Customer state"
// @Param is_active query boolean false "Active status"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Param sort_by query string false "Sort by field" Enums(name, email, created_at) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.ListCustomersResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/customers/search [get]
// @Security BearerAuth
func (uc *CustomerUseCase) Search(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Customer search attempt started", map[string]interface{}{
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

	var request entities.SearchCustomerRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind query parameters", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid query parameters")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Set defaults
	if request.Page < 1 {
		request.Page = 1
	}
	if request.PageSize < 1 || request.PageSize > 100 {
		request.PageSize = 10
	}
	if request.SortBy == "" {
		request.SortBy = "created_at"
	}
	if request.SortDir == "" {
		request.SortDir = "desc"
	}

	offset := (request.Page - 1) * request.PageSize

	// Build filters map
	filters := make(map[string]interface{})
	if request.Name != "" {
		filters["name"] = request.Name
	}
	if request.Email != "" {
		filters["email"] = request.Email
	}
	if request.Phone != "" {
		filters["phone"] = request.Phone
	}
	if request.Document != "" {
		filters["document"] = request.Document
	}
	if request.City != "" {
		filters["city"] = request.City
	}
	if request.State != "" {
		filters["state"] = request.State
	}
	if request.IsActive != nil {
		filters["is_active"] = *request.IsActive
	}
	if request.IDFilter != nil {
		filters["id"] = *request.IDFilter
	}
	filters["sort_by"] = request.SortBy
	filters["sort_dir"] = request.SortDir

	// Search customers
	customers, total, err := uc.repository.SearchCustomers(ctx, userID, admin, filters, request.PageSize, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to search customers", map[string]interface{}{
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

	totalPages := (total + request.PageSize - 1) / request.PageSize

	response := entities.ListCustomersResponse{
		Data:       customerResponses,
		Total:      total,
		Page:       request.Page,
		PageSize:   request.PageSize,
		TotalPages: totalPages,
	}

	uc.logger.Info(ctx, "Customer search completed successfully", map[string]interface{}{
		"count": len(customers),
		"total": total,
		"page":  request.Page,
	})

	c.JSON(http.StatusOK, response)
}
