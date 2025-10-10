package usecases

import (
	"net/http"
	"strconv"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/roles"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Search handles searching filaments with filters
// @Summary Search Filaments
// @Schemes
// @Description Search 3D printing filaments with various filters
// @Tags Filaments
// @Accept json
// @Produce json
// @Param name query string false "Filter by name (partial match)"
// @Param brand_id query string false "Filter by brand ID (UUID)"
// @Param material_id query string false "Filter by material ID (UUID)"
// @Param color_type query string false "Filter by color type (solid, gradient, duo, rainbow)"
// @Param diameter query number false "Filter by diameter (exact match)"
// @Param min_price query number false "Minimum price per kg"
// @Param max_price query number false "Maximum price per kg"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} filamentEntities.FindAllFilamentsResponse "Successfully retrieved filaments"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/search [get]
// @Security Bearer
func (uc *FilamentUseCase) Search(c *gin.Context) {
	ctx := c.Request.Context()

	// Log filament search attempt
	uc.logger.Info(ctx, "Filament search attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	// Extract user data from context
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok {
		appError := coreErrors.UsecaseError("Invalid user ID in context")
		httpError := appError.ToHTTPError()
		uc.logger.Error(ctx, "Invalid user ID in context", map[string]interface{}{
			"error": "user_id not found or invalid type",
		})
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if user is admin
	userRole, _ := c.Get("user_role")
	userRoleStr, _ := userRole.(string)
	isAdmin := userRoleStr == roles.AdminRole

	// Parse pagination parameters
	page := 1
	limit := 20

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Build search filters
	filters := make(map[string]interface{})

	if name := c.Query("name"); name != "" {
		filters["name"] = name
	}

	if brandIDStr := c.Query("brand_id"); brandIDStr != "" {
		if brandID, err := uuid.Parse(brandIDStr); err == nil {
			filters["brand_id"] = brandID
		} else {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid brand_id format")
			uc.logger.Error(ctx, "Invalid brand_id in search", map[string]interface{}{
				"brand_id": brandIDStr,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	if materialIDStr := c.Query("material_id"); materialIDStr != "" {
		if materialID, err := uuid.Parse(materialIDStr); err == nil {
			filters["material_id"] = materialID
		} else {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid material_id format")
			uc.logger.Error(ctx, "Invalid material_id in search", map[string]interface{}{
				"material_id": materialIDStr,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	if colorType := c.Query("color_type"); colorType != "" {
		colorTypeEnum := filamentEntities.ColorType(colorType)
		if !colorTypeEnum.IsValid() {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid color_type value")
			uc.logger.Error(ctx, "Invalid color_type in search", map[string]interface{}{
				"color_type": colorType,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
		filters["color_type"] = colorType
	}

	if diameterStr := c.Query("diameter"); diameterStr != "" {
		if diameter, err := strconv.ParseFloat(diameterStr, 64); err == nil {
			filters["diameter"] = diameter
		} else {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid diameter format")
			uc.logger.Error(ctx, "Invalid diameter in search", map[string]interface{}{
				"diameter": diameterStr,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			filters["min_price"] = minPrice
		} else {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid min_price format")
			uc.logger.Error(ctx, "Invalid min_price in search", map[string]interface{}{
				"min_price": minPriceStr,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			filters["max_price"] = maxPrice
		} else {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid max_price format")
			uc.logger.Error(ctx, "Invalid max_price in search", map[string]interface{}{
				"max_price": maxPriceStr,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Search filaments
	filaments, total, err := uc.repository.SearchFilaments(ctx, userIDStr, isAdmin, filters, limit, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to search filaments", map[string]interface{}{
			"error":   err.Error(),
			"filters": filters,
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful search
	uc.logger.Info(ctx, "Filaments search completed successfully", map[string]interface{}{
		"total_found": total,
		"returned":    len(filaments),
		"page":        page,
		"limit":       limit,
		"filters":     filters,
	})

	// Build response with related data
	responses := make([]filamentEntities.FilamentResponse, len(filaments))
	for i, filament := range filaments {
		responses[i] = filamentEntities.FilamentResponse{
			FilamentEntity: filament,
		}

		// Fetch brand information
		if brandInfo, err := uc.repository.GetBrandInfo(ctx, filament.BrandID); err == nil {
			responses[i].Brand = brandInfo
		}

		// Fetch material information
		if materialInfo, err := uc.repository.GetMaterialInfo(ctx, filament.MaterialID); err == nil {
			responses[i].Material = materialInfo
		}
	}

	totalPages := (total + limit - 1) / limit
	response := filamentEntities.FindAllFilamentsResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}
