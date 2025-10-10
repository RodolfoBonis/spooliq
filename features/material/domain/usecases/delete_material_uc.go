package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete handles deleting a material by ID
// @Summary Delete Material
// @Schemes
// @Description Delete a 3D printing material by its ID
// @Tags Materials
// @Param id path string true "Material ID" format(uuid)
// @Success 204 "Successfully deleted material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /materials/{id} [delete]
// @Security Bearer
func (uc *MaterialUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log material deletion attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Material deletion attempt started", map[string]interface{}{
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

	// Check if material exists before deletion
	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Material not found")
			httpError := appError.ToHTTPError()

			// Enhanced logging with automatic trace correlation
			uc.logger.Error(ctx, "Material not found for deletion", map[string]interface{}{
				"material_id": id,
				"error":       err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to check material existence for deletion", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Perform the deletion
	if err := uc.repository.Delete(id); err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to delete material", map[string]interface{}{
			"material_id":   id,
			"material_name": material.Name,
			"error":         err.Error(),
		})

		appError := coreErrors.UsecaseError("Failed to delete material")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful deletion with automatic trace correlation
	uc.logger.Info(ctx, "Material deleted successfully", map[string]interface{}{
		"material_id":   material.ID,
		"material_name": material.Name,
	})

	c.Status(http.StatusNoContent)
}
