package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving a paginated, searchable list of brands.
// @Summary Find All Brands
// @Description List filament brands for the organization, paginated and searchable.
// @Tags Brands
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on the brand name"
// @Param sort_by query string false "Sort field" Enums(name, created_at) default(name)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(asc)
// @Success 200 {object} entities.FindAllBrandsResponse "Paginated list of brands"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /brands [get]
// @Security BearerAuth
func (uc *BrandUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   brandSortWhitelist,
		DefaultSort:     "name",
		TieBreaker:      "id",
	})

	// Catalog lists default to ascending order by name when the client does not
	// specify a direction, matching the previous "name ASC" behavior.
	order := q.OrderClause()
	if c.Query("sort_dir") == "" {
		order = q.OrderClauseDir("asc")
	}

	brands, total, err := uc.repository.FindAll(organizationID, q.Search, order, q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve brands", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Brands retrieved successfully", map[string]interface{}{"total_brands": total})

	c.JSON(http.StatusOK, helpers.NewPage(brands, total, q))
}
