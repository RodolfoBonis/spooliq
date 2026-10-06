package usecases

import (
	"net/http"
	"strconv"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Search handles searching filaments with structured filters plus free-text
// search, pagination and whitelisted sorting.
// @Summary Search Filaments
// @Description Search 3D printing filaments with filters, pagination and sorting.
// @Tags Filaments
// @Accept json
// @Produce json
// @Param q query string false "Case-insensitive search on filament/brand/material name"
// @Param name query string false "Alias of q (case-insensitive name search)"
// @Param brand_id query string false "Filter by brand ID (UUID)"
// @Param material_id query string false "Filter by material ID (UUID)"
// @Param color_type query string false "Filter by color type (solid, gradient, duo, rainbow, ...)"
// @Param diameter query number false "Filter by diameter (exact match)"
// @Param min_price query number false "Minimum price per kg"
// @Param max_price query number false "Maximum price per kg"
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param sort_by query string false "Sort field" Enums(name, created_at, price_per_kg) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.FindAllFilamentsResponse "Paginated list of filaments"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments/search [get]
// @Security BearerAuth
func (uc *FilamentUseCase) Search(c *gin.Context) {
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
	})

	// `name` is a legacy alias of `q`; q wins when both are present.
	search := q.Search
	if search == "" {
		search = c.Query("name")
	}

	filters, apiErr := parseFilamentFilters(c)
	if apiErr != nil {
		uc.logger.Error(ctx, "Invalid filament search filter", map[string]interface{}{"code": apiErr.Code})
		coreErrors.Respond(c, apiErr)
		return
	}

	filaments, total, err := uc.repository.SearchFilaments(ctx, organizationID, filters, search, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to search filaments", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	responses := uc.buildFilamentResponses(ctx, filaments)

	uc.logger.Info(ctx, "Filaments search completed successfully", map[string]interface{}{"total_found": total, "returned": len(responses)})

	c.JSON(http.StatusOK, helpers.NewPage(responses, total, q))
}

// parseFilamentFilters reads the structured search filters from the query
// string, validating each and returning a stable APIError code on bad input.
func parseFilamentFilters(c *gin.Context) (map[string]interface{}, *coreErrors.APIError) {
	filters := make(map[string]interface{})

	if brandIDStr := c.Query("brand_id"); brandIDStr != "" {
		brandID, err := uuid.Parse(brandIDStr)
		if err != nil {
			return nil, coreErrors.BadRequest("invalid_brand_id", "ID de marca inválido")
		}
		filters["brand_id"] = brandID
	}

	if materialIDStr := c.Query("material_id"); materialIDStr != "" {
		materialID, err := uuid.Parse(materialIDStr)
		if err != nil {
			return nil, coreErrors.BadRequest("invalid_material_id", "ID de material inválido")
		}
		filters["material_id"] = materialID
	}

	if colorType := c.Query("color_type"); colorType != "" {
		if !filamentEntities.ColorType(colorType).IsValid() {
			return nil, coreErrors.BadRequest("invalid_color_type", "Tipo de cor inválido")
		}
		filters["color_type"] = colorType
	}

	if diameterStr := c.Query("diameter"); diameterStr != "" {
		diameter, err := strconv.ParseFloat(diameterStr, 64)
		if err != nil {
			return nil, coreErrors.BadRequest("invalid_diameter", "Diâmetro inválido")
		}
		filters["diameter"] = diameter
	}

	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		minPrice, err := strconv.ParseFloat(minPriceStr, 64)
		if err != nil {
			return nil, coreErrors.BadRequest("invalid_min_price", "Preço mínimo inválido")
		}
		filters["min_price"] = minPrice
	}

	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
		if err != nil {
			return nil, coreErrors.BadRequest("invalid_max_price", "Preço máximo inválido")
		}
		filters["max_price"] = maxPrice
	}

	return filters, nil
}
