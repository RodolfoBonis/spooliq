package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving a paginated, searchable list of filaments.
// @Summary Find All Filaments
// @Description List 3D printing filaments for the organization, paginated and searchable.
// @Tags Filaments
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on filament/brand/material name"
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

	filaments, total, err := uc.repository.FindAll(ctx, organizationID, q.Search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve filaments", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	responses := uc.buildFilamentResponses(ctx, organizationID, filaments)

	uc.logger.Info(ctx, "Filaments retrieved successfully", map[string]interface{}{"total_filaments": total, "returned": len(responses)})

	c.JSON(http.StatusOK, helpers.NewPage(responses, total, q))
}
