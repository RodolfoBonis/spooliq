package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create handles creating a new material.
// @Summary Create Material
// @Description Create a new 3D printing material
// @Tags Materials
// @Accept json
// @Produce json
// @Param request body entities.UpsertMaterialRequestEntity true "Material data"
// @Success 201 {object} entities.MaterialEntity "Successfully created material"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /materials [post]
// @Security BearerAuth
func (uc *MaterialUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	var request entities.UpsertMaterialRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid material creation payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Material creation validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	exists, err := uc.repository.Exists(request.Name, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check material name existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if exists {
		uc.logger.Warning(ctx, "Material creation failed: name already exists", map[string]interface{}{"name": request.Name})
		coreErrors.Respond(c, coreErrors.Conflict("material_name_taken", "Já existe um material com este nome"))
		return
	}

	material := entities.MaterialEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           request.Name,
		Description:    request.Description,
		TempTable:      request.TempTable,
		TempExtruder:   request.TempExtruder,
	}

	if err := uc.repository.Create(&material); err != nil {
		uc.logger.Error(ctx, "Failed to create material", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Material created successfully", map[string]interface{}{"material_id": material.ID})

	c.JSON(http.StatusCreated, material)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityMaterial,
		EntityID:       material.ID.String(),
		EntityName:     material.Name,
		Description:    "Material created: " + material.Name,
	})
}
