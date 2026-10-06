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

// Delete handles deleting a brand by ID.
// @Summary Delete Brand
// @Description Delete a filament brand by its ID
// @Tags Brands
// @Param id path string true "Brand ID" format(uuid)
// @Success 204 "Successfully deleted brand"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /brands/{id} [delete]
// @Security BearerAuth
func (uc *BrandUseCase) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid brand ID", map[string]interface{}{"brand_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_brand_id", "ID de marca inválido"))
		return
	}

	brand, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Brand not found for deletion", map[string]interface{}{"brand_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("brand_not_found", "Marca não encontrada"))
			return
		}
		uc.logger.Error(ctx, "Failed to check brand existence for deletion", map[string]interface{}{"brand_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := uc.repository.Delete(id); err != nil {
		// A RESTRICT foreign key (filaments reference the brand) surfaces as a
		// 23503 violation; report it as a clean 409 instead of a 500.
		if database.IsForeignKeyViolation(err) {
			uc.logger.Warning(ctx, "Brand deletion blocked: brand in use", map[string]interface{}{"brand_id": id})
			coreErrors.Respond(c, coreErrors.Conflict("brand_in_use", "Marca em uso e não pode ser removida"))
			return
		}
		uc.logger.Error(ctx, "Failed to delete brand", map[string]interface{}{"brand_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Brand deleted successfully", map[string]interface{}{"brand_id": brand.ID})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityBrand,
		EntityID:       brand.ID.String(),
		EntityName:     brand.Name,
		Description:    "Brand deleted: " + brand.Name,
	})
}
