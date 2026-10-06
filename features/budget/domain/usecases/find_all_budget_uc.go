package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll lists budgets for the organization with pagination, filtering and sorting.
// @Summary List budgets
// @Description List budgets (paginated). Supports free-text name search (q), and
// @Description filtering by status, customer_id and a created_at range (from/to,
// @Description YYYY-MM-DD or RFC3339, inclusive). Sortable by created_at, name,
// @Description total_cost and status (default created_at desc).
// @Tags budgets
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size (max 100)" default(20)
// @Param q query string false "Free-text search on budget name (case-insensitive)"
// @Param status query string false "Filter by status" Enums(draft, sent, approved, rejected, printing, completed)
// @Param customer_id query string false "Filter by customer UUID"
// @Param from query string false "Created-at lower bound (YYYY-MM-DD or RFC3339, inclusive)"
// @Param to query string false "Created-at upper bound (YYYY-MM-DD or RFC3339, inclusive)"
// @Param sort_by query string false "Sort field" Enums(created_at, name, total_cost, status) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} helpers.Page[entities.BudgetResponse]
// @Failure 400 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets [get]
// @Security BearerAuth
func (uc *BudgetUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	listQuery := budgetListQuery(c)

	filters, apiErr := parseBudgetFilters(c, listQuery.Search)
	if apiErr != nil {
		coreErrors.Respond(c, apiErr)
		return
	}

	budgets, total, err := uc.budgetRepository.SearchBudgets(ctx, organizationID, filters, listQuery.OrderClause(), listQuery.Limit(), listQuery.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budgets", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	budgetResponses, err := uc.buildBudgetListResponses(ctx, budgets, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to build budget list response", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	page := helpers.NewPage(budgetResponses, int64(total), listQuery)

	uc.logger.Info(ctx, "Budgets retrieved successfully", map[string]interface{}{
		"count": len(budgetResponses),
		"total": total,
		"page":  listQuery.Page,
	})

	c.JSON(http.StatusOK, page)
}
