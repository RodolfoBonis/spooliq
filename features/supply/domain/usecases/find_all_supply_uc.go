package usecases

import (
	"net/http"
	"strconv"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving a paginated, searchable list of supplies.
// @Summary Find All Supplies
// @Description List non-printed supplies (insumos) for the organization, paginated, searchable and filterable by active state.
// @Tags Supplies
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on the supply name"
// @Param search query string false "Alias of q (case-insensitive name search)"
// @Param is_active query boolean false "Filter by active state (true/false). Omitted returns all."
// @Param sort_by query string false "Sort field" Enums(name, unit_cost, created_at) default(name)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(asc)
// @Success 200 {object} entities.FindAllSuppliesResponse "Paginated list of supplies"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /supplies [get]
// @Security BearerAuth
func (uc *SupplyUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   supplySortWhitelist,
		DefaultSort:     "name",
		TieBreaker:      "id",
	})

	// Catalog lists default to ascending order by name when the client does not
	// specify a direction, matching the brand/material behavior.
	order := q.OrderClause()
	if c.Query("sort_dir") == "" {
		order = q.OrderClauseDir("asc")
	}

	// `q` is the convention used by the other catalog lists; `search` is accepted as
	// an alias (q wins when both are present) so the desktop app can use either.
	search := q.Search
	if search == "" {
		search = c.Query("search")
	}

	// is_active is an optional tri-state filter: absent means "all"; "true"/"false"
	// filters by the active flag. An unparseable value is a 400.
	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		v, perr := strconv.ParseBool(raw)
		if perr != nil {
			uc.logger.Error(ctx, "Invalid is_active filter", map[string]interface{}{"is_active": raw})
			coreErrors.Respond(c, coreErrors.BadRequest("invalid_is_active", "Filtro is_active inválido (use true ou false)"))
			return
		}
		isActive = &v
	}

	supplies, total, err := uc.repository.FindAll(organizationID, search, isActive, order, q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve supplies", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Supplies retrieved successfully", map[string]interface{}{"total_supplies": total})

	c.JSON(http.StatusOK, helpers.NewPage(supplies, total, q))
}
