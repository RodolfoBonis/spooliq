package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
)

// Search searches for customers with structured filters plus free-text search,
// pagination and whitelisted sorting.
// @Summary Search customers
// @Description Search customers with filters, pagination and sorting.
// @Tags customers
// @Accept json
// @Produce json
// @Param q query string false "Case-insensitive search on name/email/phone/document"
// @Param name query string false "Filter by name (partial, case-insensitive)"
// @Param email query string false "Filter by email (partial, case-insensitive)"
// @Param phone query string false "Filter by phone (partial)"
// @Param document query string false "Filter by document/CPF/CNPJ (partial)"
// @Param city query string false "Filter by city (partial, case-insensitive)"
// @Param state query string false "Filter by state (partial, case-insensitive)"
// @Param is_active query boolean false "Filter by active status"
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param sort_by query string false "Sort field" Enums(name, email, created_at) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.ListCustomersResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers/search [get]
// @Security BearerAuth
func (uc *CustomerUseCase) Search(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	var request entities.SearchCustomerRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind query parameters", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}
	// Enforce the sort_by/sort_dir enums (and any other validate tags).
	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Invalid customer search parameters", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   customerSortWhitelist,
		DefaultSort:     "created_at",
		TieBreaker:      "id",
	})

	// Build structured filters from the bound request.
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

	customers, total, err := uc.repository.SearchCustomers(ctx, organizationID, filters, q.Search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to search customers", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	responses := uc.buildCustomerResponses(ctx, customers)

	uc.logger.Info(ctx, "Customer search completed successfully", map[string]interface{}{"count": len(responses), "total": total})

	c.JSON(http.StatusOK, helpers.NewPage(responses, total, q))
}
