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

// Delete handles deleting a brand by ID
// @Summary Delete Brand
// @Schemes
// @Description Delete a filament brand by its ID
// @Tags Brands
// @Param id path string true "Brand ID" format(uuid)
// @Success 204 "Successfully deleted brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands/{id} [delete]
// @Security Bearer
func (uc *BrandUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log brand deletion attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Brand deletion attempt started", map[string]interface{}{
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

	// Check if brand exists before deletion
	brand, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Brand not found")
			httpError := appError.ToHTTPError()

			// Enhanced logging with automatic trace correlation
			uc.logger.Error(ctx, "Brand not found for deletion", map[string]interface{}{
				"brand_id": id,
				"error":    err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to check brand existence for deletion", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Perform the deletion
	if err := uc.repository.Delete(id); err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to delete brand", map[string]interface{}{
			"brand_id":   id,
			"brand_name": brand.Name,
			"error":      err.Error(),
		})

		appError := coreErrors.UsecaseError("Failed to delete brand")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful deletion with automatic trace correlation
	uc.logger.Info(ctx, "Brand deleted successfully", map[string]interface{}{
		"brand_id":   brand.ID,
		"brand_name": brand.Name,
	})

	c.Status(http.StatusNoContent)
}
