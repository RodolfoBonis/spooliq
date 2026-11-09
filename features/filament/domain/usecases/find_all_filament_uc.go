package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving a paginated, searchable list of filaments.
//
// It honours the SAME structured filters as /filaments/search (brand_id,
// material_id, color_type, diameter, min_price, max_price) by reusing
// parseFilamentFilters and the repository search path, so clients get consistent
// behavior regardless of which endpoint they hit. Filters are optional; with none
// supplied it behaves like a plain paginated list.
// @Summary Find All Filaments
// @Description List 3D printing filaments for the organization, paginated, searchable and filterable.
// @Tags Filaments
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on filament/brand/material name"
// @Param name query string false "Alias of q (case-insensitive name search)"
// @Param brand_id query string false "Filter by brand ID (UUID)"
// @Param material_id query string false "Filter by material ID (UUID)"
// @Param color_type query string false "Filter by color type (solid, gradient, duo, rainbow, ...)"
// @Param diameter query number false "Filter by diameter (exact match)"
// @Param min_price query number false "Minimum price per kg"
// @Param max_price query number false "Maximum price per kg"
// @Param low_stock query boolean false "Only filaments at or below their low-stock threshold"
// @Param sort_by query string false "Sort field" Enums(name, created_at, price_per_kg) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.FindAllFilamentsResponse "Paginated list of filaments"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments [get]
// @Security BearerAuth
func (uc *FilamentUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   filamentSortWhitelist,
		DefaultSort:     "created_at",
		TieBreaker:      "filaments.id",
	})

	// `name` is a legacy alias of `q`; q wins when both are present. Mirrors Search.
	search := q.Search
	if search == "" {
		search = c.Query("name")
	}

	filters, apiErr := parseFilamentFilters(c)
	if apiErr != nil {
		uc.logger.Error(ctx, "Invalid filament filter", map[string]interface{}{"code": apiErr.Code})
		coreErrors.Respond(c, apiErr)
		return
	}

	filaments, total, err := uc.repository.SearchFilaments(ctx, organizationID, filters, search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve filaments", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	responses := uc.buildFilamentResponses(ctx, organizationID, filaments)

	uc.logger.Info(ctx, "Filaments retrieved successfully", map[string]interface{}{"total_filaments": total, "returned": len(responses)})

	c.JSON(http.StatusOK, helpers.NewPage(responses, total, q))
}
