package usecases

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/helpers"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a specific material by ID
// @Summary Get Material by ID
// @Schemes
// @Description Get a specific 3D printing material by its ID
// @Tags Materials
// @Accept json
// @Produce json
// @Param id path string true "Material ID" format(uuid)
// @Success 200 {object} entities.FindByIDMaterialResponse "Successfully retrieved material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /materials/{id} [get]
// @Security Bearer
func (uc *MaterialUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	// Log material retrieval attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Material retrieval attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid material ID", map[string]interface{}{
			"material_id": idParam,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError("Invalid material ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Material not found")
			httpError := appError.ToHTTPError()

			// Enhanced logging with automatic trace correlation
			uc.logger.Error(ctx, "Material not found", map[string]interface{}{
				"material_id": id,
				"error":       err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to retrieve material", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful retrieval with automatic trace correlation
	uc.logger.Info(ctx, "Material retrieved successfully", map[string]interface{}{
		"material_id":   material.ID,
		"material_name": material.Name,
	})

	c.JSON(http.StatusOK, entities.FindByIDMaterialResponse{Data: *material})
}
