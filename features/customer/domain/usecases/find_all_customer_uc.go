package usecases

import (
	"context"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindAll retrieves a paginated, searchable list of customers.
// @Summary List customers
// @Description List customers for the organization, paginated and searchable.
// @Tags customers
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on name/email/phone/document"
// @Param sort_by query string false "Sort field" Enums(name, email, created_at) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.ListCustomersResponse
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /customers [get]
// @Security BearerAuth
func (uc *CustomerUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   customerSortWhitelist,
		DefaultSort:     "created_at",
		TieBreaker:      "id",
	})

	customers, total, err := uc.repository.FindAll(ctx, organizationID, q.Search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve customers", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	responses := uc.buildCustomerResponses(ctx, customers)

	uc.logger.Info(ctx, "Customers retrieved successfully", map[string]interface{}{"count": len(responses), "total": total})

	c.JSON(http.StatusOK, helpers.NewPage(responses, total, q))
}

// buildCustomerResponses assembles CustomerResponse values for a page of
// customers, batch-loading the budget count and status-filtered total in a
// single grouped query regardless of row count. This is the N+1 fix for the
// list/search endpoints, which previously issued one or two queries per row.
func (uc *CustomerUseCase) buildCustomerResponses(ctx context.Context, customers []*entities.CustomerEntity) []entities.CustomerResponse {
	responses := make([]entities.CustomerResponse, len(customers))

	ids := make([]uuid.UUID, 0, len(customers))
	for _, cust := range customers {
		ids = append(ids, cust.ID)
	}

	stats, err := uc.repository.GetBudgetStatsByCustomers(ctx, ids, budgetTotalStatuses)
	if err != nil {
		uc.logger.Error(ctx, "Failed to batch-load customer budget stats", map[string]interface{}{"error": err.Error()})
		stats = map[uuid.UUID]entities.CustomerBudgetStats{}
	}

	for i, cust := range customers {
		resp := entities.CustomerResponse{Customer: cust}
		total := int64(0)
		if s, ok := stats[cust.ID]; ok {
			resp.BudgetCount = int(s.Count)
			total = s.Total
		}
		resp.TotalBudgets = &total
		responses[i] = resp
	}

	return responses
}
