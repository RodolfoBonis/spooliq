package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete soft-deletes a 3D model, organization-scoped.
//
// It deliberately does NOT remove the CDN objects (file/thumbnail). Budgets may
// reference this model (budget_items.model_3d_id), and the row is only
// soft-deleted, so the underlying file must remain retrievable. CDNService.DeleteFile
// is kept available for a future hard-delete/garbage-collection path.
// @Summary Delete 3D Model
// @Description Soft-delete a 3D model by its ID. The stored file is intentionally kept.
// @Tags models3d
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Success 204 "Successfully deleted 3D model"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d/{id} [delete]
// @Security BearerAuth
func (uc *Model3DUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, errInvalidModel3DID())
		return
	}

	model, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve 3D model for deletion", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := uc.repository.Delete(ctx, id, organizationID); err != nil {
		uc.logger.Error(ctx, "Failed to delete 3D model", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	uc.logger.Info(ctx, "3D model deleted successfully", map[string]interface{}{"model_id": model.ID, "model_name": model.Name})

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
