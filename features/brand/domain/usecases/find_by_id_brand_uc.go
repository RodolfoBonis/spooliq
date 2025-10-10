package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a specific brand by ID
// @Summary Get Brand by ID
// Schemes
// @Description Get a specific filament brand by its ID
// @Tags Brands
// @Accept json
// @Produce json
// @Param id path string true "Brand ID" format(uuid)
// @Success 200 {object} entities.FindByIDBrandResponse "Successfully retrieved brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands/{id} [get]
// @Security Bearer
func (uc *BrandUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log brand retrieval attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Brand retrieval attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid brand ID", map[string]interface{}{
			"brand_id": idParam,
			"error":    err.Error(),
		})

		appError := coreErrors.UsecaseError("Invalid brand ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	brand, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Brand not found")
			httpError := appError.ToHTTPError()

			// Enhanced logging with automatic trace correlation
			uc.logger.Error(ctx, "Brand not found", map[string]interface{}{
				"brand_id": id,
				"error":    err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to retrieve brand", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful retrieval with automatic trace correlation
	uc.logger.Info(ctx, "Brand retrieved successfully", map[string]interface{}{
		"brand_id":   brand.ID,
		"brand_name": brand.Name,
	})

	c.JSON(http.StatusOK, entities.FindByIDBrandResponse{Data: *brand})
}
