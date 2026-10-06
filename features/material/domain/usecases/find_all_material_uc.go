package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving a paginated, searchable list of materials.
// @Summary Find All Materials
// @Description List 3D printing materials for the organization, paginated and searchable.
// @Tags Materials
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on the material name"
// @Param sort_by query string false "Sort field" Enums(name, created_at) default(name)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(asc)
// @Success 200 {object} entities.FindAllMaterialsResponse "Paginated list of materials"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /materials [get]
// @Security BearerAuth
func (uc *MaterialUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   materialSortWhitelist,
		DefaultSort:     "name",
	})

	order := q.OrderClause()
	if c.Query("sort_dir") == "" {
		order = q.SortColumn + " asc"
	}

	materials, total, err := uc.repository.FindAll(organizationID, q.Search, order, q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve materials", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Materials retrieved successfully", map[string]interface{}{"total_materials": total})

	c.JSON(http.StatusOK, helpers.NewPage(materials, total, q))
}
