package usecases

import (
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete handles soft-deleting a 3D model.
// @Summary Delete 3D Model
// Schemes
// @Description Soft-delete a 3D model by its ID
// @Tags 3D Models
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Success 204 "Successfully deleted 3D model"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /models3d/{id} [delete]
// @Security Bearer
func (uc *Model3DUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		uc.logger.Error(ctx, "Invalid model ID", map[string]interface{}{
			"model_id": idParam,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid model ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	model, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("3D model not found")
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve 3D model for deletion", map[string]interface{}{
			"model_id": id,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.repository.Delete(id); err != nil {
		uc.logger.Error(ctx, "Failed to delete 3D model", map[string]interface{}{
			"model_id":   id,
			"model_name": model.Name,
			"error":      err.Error(),
		})
		httpError := coreErrors.NewHTTPError(http.StatusInternalServerError, "Failed to delete 3D model")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Clean up CDN files (best-effort: log errors but don't fail the request)
	if model.FileURL != "" {
		if err := uc.cdnService.DeleteFile(ctx, model.FileURL); err != nil {
			uc.logger.Warning(ctx, "Failed to delete model file from CDN", map[string]interface{}{
				"model_id": model.ID,
				"file_url": model.FileURL,
				"error":    err.Error(),
			})
		}
	}
	if model.ThumbnailURL != nil && *model.ThumbnailURL != "" {
		if err := uc.cdnService.DeleteFile(ctx, *model.ThumbnailURL); err != nil {
			uc.logger.Warning(ctx, "Failed to delete thumbnail from CDN", map[string]interface{}{
				"model_id":      model.ID,
				"thumbnail_url": *model.ThumbnailURL,
				"error":         err.Error(),
			})
		}
	}

	uc.logger.Info(ctx, "3D model deleted successfully", map[string]interface{}{
		"model_id":   model.ID,
		"model_name": model.Name,
	})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityModel3D,
		EntityID:       model.ID.String(),
		EntityName:     model.Name,
		Description:    "3D Model deleted: " + model.Name,
	})
}
