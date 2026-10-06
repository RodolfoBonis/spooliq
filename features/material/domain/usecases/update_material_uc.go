package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating an existing material.
// @Summary Update Material
// @Description Update an existing 3D printing material
// @Tags Materials
// @Accept json
// @Produce json
// @Param id path string true "Material ID" format(uuid)
// @Param request body entities.UpsertMaterialRequestEntity true "Material data"
// @Success 200 {object} entities.MaterialEntity "Successfully updated material"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /materials/{id} [put]
// @Security BearerAuth
func (uc *MaterialUseCase) Update(c *gin.Context) {
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

	var request entities.UpsertMaterialRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid material update payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Material update validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Material not found for update", map[string]interface{}{"material_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("material_not_found", "Material não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to get material for update", map[string]interface{}{"material_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Enforce unique material name within the organization (excluding this one).
	if request.Name != material.Name {
		exists, err := uc.repository.Exists(request.Name, organizationID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check material name existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
			coreErrors.Respond(c, err)
			return
		}
		if exists {
			uc.logger.Warning(ctx, "Material update failed: name already exists", map[string]interface{}{"name": request.Name})
			coreErrors.Respond(c, coreErrors.Conflict("material_name_taken", "Já existe um material com este nome"))
			return
		}
	}

	material.Name = request.Name
	material.Description = request.Description
	material.TempTable = request.TempTable
	material.TempExtruder = request.TempExtruder

	if err := uc.repository.Update(material); err != nil {
		uc.logger.Error(ctx, "Failed to update material", map[string]interface{}{"material_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Material updated successfully", map[string]interface{}{"material_id": material.ID})

	c.JSON(http.StatusOK, material)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityMaterial,
		EntityID:       material.ID.String(),
		EntityName:     material.Name,
		Description:    "Material updated: " + material.Name,
	})
}
