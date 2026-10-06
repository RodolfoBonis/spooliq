package usecases

import (
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// model3dSortWhitelist maps the public sort names to safe column expressions. Only
// names present here are ever accepted (injection-safe boundary).
var model3dSortWhitelist = map[string]string{
	"name":            "name",
	"created_at":      "created_at",
	"file_size_bytes": "file_size_bytes",
}

// FindAll handles listing 3D models with pagination, search, filters and sorting.
// @Summary List 3D Models
// @Description List the organization's 3D models, paginated, searchable and filterable.
// @Tags models3d
// @Accept json
// @Produce json
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param q query string false "Case-insensitive search on name/tags"
// @Param format query string false "Filter by file format" Enums(.stl, .3mf)
// @Param customer_id query string false "Filter by customer ID" format(uuid)
// @Param sort_by query string false "Sort field" Enums(name, created_at, file_size_bytes) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.ListModel3DResponse "Paginated list of 3D models"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d [get]
// @Security BearerAuth
func (uc *Model3DUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   model3dSortWhitelist,
		DefaultSort:     "created_at",
		TieBreaker:      "id",
	})

	filters, apiErr := parseModel3DFilters(c)
	if apiErr != nil {
		uc.logger.Error(ctx, "Invalid 3D model filter", map[string]interface{}{"code": apiErr.Code})
		coreErrors.Respond(c, apiErr)
		return
	}

	models, total, err := uc.repository.FindAll(ctx, organizationID, filters, q.Search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to list 3D models", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "3D models retrieved successfully", map[string]interface{}{"count": len(models), "total": total})

	c.JSON(http.StatusOK, helpers.NewPage(models, total, q))
}

// parseModel3DFilters reads the structured list filters, validating each and
// returning a stable APIError on bad input.
func parseModel3DFilters(c *gin.Context) (repositories.Model3DFilters, *coreErrors.APIError) {
	var filters repositories.Model3DFilters

	if format := strings.ToLower(strings.TrimSpace(c.Query("format"))); format != "" {
		if !allowedExtensions[format] {
			return filters, errInvalidFormat()
		}
		filters.Format = format
	}

	if cidStr := strings.TrimSpace(c.Query("customer_id")); cidStr != "" {
		parsed, err := uuid.Parse(cidStr)
		if err != nil {
			return filters, errInvalidCustomerID()
		}
		filters.CustomerID = &parsed
	}

	return filters, nil
}
