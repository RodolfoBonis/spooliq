package usecases

import (
	"errors"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/database"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete handles deleting a material by ID.
// @Summary Delete Material
// @Description Delete a 3D printing material by its ID
// @Tags Materials
// @Param id path string true "Material ID" format(uuid)
// @Success 204 "Successfully deleted material"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /materials/{id} [delete]
// @Security BearerAuth
func (uc *MaterialUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid material ID", map[string]interface{}{"material_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_material_id", "ID de material inválido"))
		return
	}

	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Material not found for deletion", map[string]interface{}{"material_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("material_not_found", "Material não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to check material existence for deletion", map[string]interface{}{"material_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := uc.repository.Delete(id); err != nil {
		// A RESTRICT foreign key (filaments reference the material) surfaces as a
		// 23503 violation; report it as a clean 409 instead of a 500.
		if database.IsForeignKeyViolation(err) {
			uc.logger.Warning(ctx, "Material deletion blocked: material in use", map[string]interface{}{"material_id": id})
			coreErrors.Respond(c, coreErrors.Conflict("material_in_use", "Material em uso e não pode ser removido"))
			return
		}
		uc.logger.Error(ctx, "Failed to delete material", map[string]interface{}{"material_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Material deleted successfully", map[string]interface{}{"material_id": material.ID})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityMaterial,
		EntityID:       material.ID.String(),
		EntityName:     material.Name,
		Description:    "Material deleted: " + material.Name,
	})
}
