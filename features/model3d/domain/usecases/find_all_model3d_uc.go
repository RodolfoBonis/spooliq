package usecases

import (
	"net/http"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FindAll handles listing 3D models with pagination and filters.
// @Summary List 3D Models
// Schemes
// @Description List all 3D models with pagination and optional filters
// @Tags 3D Models
// @Accept json
// @Produce json
// @Param search query string false "Search term for model name or description"
// @Param customer_id query string false "Filter by customer ID" format(uuid)
// @Param format query string false "Filter by file format (e.g. .stl, .3mf)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(20)
// @Success 200 {object} entities.FindAllModel3DResponse "Successfully retrieved 3D models"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /models3d [get]
// @Security Bearer
func (uc *Model3DUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	// Parse query parameters
	search := c.Query("search")
	format := c.Query("format")

	page := 1
	if p := c.DefaultQuery("page", "1"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 20
	if ps := c.DefaultQuery("page_size", "20"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	var customerID *uuid.UUID
	if cidStr := c.Query("customer_id"); cidStr != "" {
		if parsed, err := uuid.Parse(cidStr); err == nil {
			customerID = &parsed
		}
	}

	filters := repositories.Model3DFilters{
		Search:     search,
		CustomerID: customerID,
		Format:     format,
		Page:       page,
		PageSize:   pageSize,
	}

	result, err := uc.repository.FindAll(organizationID, filters)
	if err != nil {
		uc.logger.Error(ctx, "Failed to list 3D models", map[string]interface{}{
			"error": err.Error(),
		})
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to list 3D models"})
		return
	}

	c.JSON(http.StatusOK, result)
}
