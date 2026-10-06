package usecases

import (
	"strings"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// budgetListPageSize is the default page size for the budget list endpoints.
const budgetListPageSize = 20

// budgetHistoryPageSize is the default page size for the status-history endpoint.
const budgetHistoryPageSize = 50

// budgetSortWhitelist maps the public sort_by names accepted on GET /budgets to the
// safe SQL column expressions used in ORDER BY. Only these names are honored, which
// is what makes the sort injection-safe (see helpers.ParseListQuery).
var budgetSortWhitelist = map[string]string{
	"created_at": "created_at",
	"name":       "LOWER(name)",
	"total_cost": "total_cost",
	"status":     "status",
}

// budgetDefaultSort is the sort applied when sort_by is omitted or not whitelisted.
const budgetDefaultSort = "created_at"

// budgetListQuery parses the shared pagination/sort params for a budget list.
func budgetListQuery(c *gin.Context) helpers.ListQuery {
	return helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: budgetListPageSize,
		SortWhitelist:   budgetSortWhitelist,
		DefaultSort:     budgetDefaultSort,
	})
}

// parseBudgetFilters reads the budget-specific list filters (free-text name search,
// status, customer_id, from/to created_at range) into a repository filter map. It
// returns a 400 *APIError for an unknown status, a malformed customer_id or an
// unparseable date so the handler can surface a stable code.
func parseBudgetFilters(c *gin.Context, search string) (map[string]interface{}, *coreErrors.APIError) {
	filters := map[string]interface{}{}

	if search != "" {
		filters["name"] = search
	}

	if status := strings.TrimSpace(c.Query("status")); status != "" {
		if !entities.IsKnownStatus(status) {
			return nil, coreErrors.BadRequest(CodeInvalidStatusFilter,
				"Status de filtro inválido. Valores aceitos: draft, sent, approved, rejected, printing, completed")
		}
		filters["status"] = status
	}

	if cid := strings.TrimSpace(c.Query("customer_id")); cid != "" {
		id, err := uuid.Parse(cid)
		if err != nil {
			return nil, coreErrors.BadRequest(CodeInvalidCustomerID, "ID de cliente inválido")
		}
		filters["customer_id"] = id
	}

	if from := strings.TrimSpace(c.Query("from")); from != "" {
		t, err := parseFilterDate(from, false)
		if err != nil {
			return nil, coreErrors.BadRequest(CodeInvalidDate, "Data inicial inválida. Use o formato YYYY-MM-DD ou RFC3339")
		}
		filters["start_date"] = t
	}

	if to := strings.TrimSpace(c.Query("to")); to != "" {
		t, err := parseFilterDate(to, true)
		if err != nil {
			return nil, coreErrors.BadRequest(CodeInvalidDate, "Data final inválida. Use o formato YYYY-MM-DD ou RFC3339")
		}
		filters["end_date"] = t
	}

	return filters, nil
}

// parseFilterDate parses a date in RFC3339 or YYYY-MM-DD. For a date-only upper
// bound (endOfDay=true) it returns the last instant of that day so the created_at
// range stays inclusive of the whole "to" day.
func parseFilterDate(raw string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return t.Add(24*time.Hour - time.Nanosecond), nil
	}
	return t, nil
}
