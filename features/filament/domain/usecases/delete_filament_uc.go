package usecases

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete handles deleting a filament (soft delete)
// @Summary Delete Filament
// @Schemes
// @Description Delete a 3D printing filament (soft delete)
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Success 204 "Successfully deleted filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [delete]
// @Security Bearer
func (uc *FilamentUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	// Log filament deletion attempt
	uc.logger.Info(ctx, "Filament deletion attempt started", map[string]interface{}{
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
	isAdmin := userRoleStr == roles.OrgAdmin

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{
			"filament_id": idParam,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError("Invalid filament ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Fetch existing filament to check ownership
	existingFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Filament not found")
			httpError := appError.ToHTTPError()

			uc.logger.Error(ctx, "Filament not found", map[string]interface{}{
				"filament_id": id,
				"error":       err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check permissions - only owner or admin can delete
	if !isAdmin && (existingFilament.OwnerUserID == nil || *existingFilament.OwnerUserID != userIDStr) {
		appError := coreErrors.UsecaseError("Access denied: you can only delete your own filaments")
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Access denied to delete filament", map[string]interface{}{
			"filament_id": id,
			"user_id":     userIDStr,
		})

		c.AbortWithStatusJSON(http.StatusForbidden, httpError)
		return
	}

	// Delete from database (soft delete)
	if err := uc.repository.Delete(ctx, id); err != nil {
		uc.logger.Error(ctx, "Failed to delete filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
			"operation":   "delete_filament",
		})

		httpError := coreErrors.NewHTTPError(http.StatusInternalServerError, "Failed to delete filament")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(ctx, "Filament deleted successfully", map[string]interface{}{
		"filament_id":   id,
		"filament_name": existingFilament.Name,
	})

	c.Status(http.StatusNoContent)
}
