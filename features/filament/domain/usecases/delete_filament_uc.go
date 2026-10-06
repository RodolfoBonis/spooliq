package usecases

import (
	"errors"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/database"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Delete handles deleting a filament (soft delete).
// @Summary Delete Filament
// @Description Delete a 3D printing filament (soft delete)
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Success 204 "Successfully deleted filament"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments/{id} [delete]
// @Security BearerAuth
func (uc *FilamentUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	userIDStr := helpers.GetUserID(c)
	if userIDStr == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("user_required", "Usuário não encontrado no contexto"))
		return
	}

	userRole, _ := c.Get("user_role")
	userRoleStr, _ := userRole.(string)
	isAdmin := userRoleStr == roles.OrgAdminRole

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{"filament_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_filament_id", "ID de filamento inválido"))
		return
	}

	existingFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Filament not found", map[string]interface{}{"filament_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("filament_not_found", "Filamento não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve filament", map[string]interface{}{"filament_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Only the owner or an admin may delete a filament.
	if !isAdmin && (existingFilament.OwnerUserID == nil || *existingFilament.OwnerUserID != userIDStr) {
		uc.logger.Warning(ctx, "Access denied to delete filament", map[string]interface{}{"filament_id": id, "user_id": userIDStr})
		coreErrors.Respond(c, coreErrors.Forbidden("filament_access_denied", "Você só pode remover seus próprios filamentos"))
		return
	}

	if err := uc.repository.Delete(ctx, id); err != nil {
		if database.IsForeignKeyViolation(err) {
			uc.logger.Warning(ctx, "Filament deletion blocked: filament in use", map[string]interface{}{"filament_id": id})
			coreErrors.Respond(c, coreErrors.Conflict("filament_in_use", "Filamento em uso e não pode ser removido"))
			return
		}
		uc.logger.Error(ctx, "Failed to delete filament", map[string]interface{}{"filament_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Filament deleted successfully", map[string]interface{}{"filament_id": id})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityFilament,
		EntityID:       id.String(),
		EntityName:     existingFilament.Name,
		CreatedAt:      time.Now(),
	})
}
