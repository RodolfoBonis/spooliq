package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating an existing brand.
// @Summary Update Brand
// @Description Update an existing filament brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param id path string true "Brand ID" format(uuid)
// @Param request body entities.UpsertBrandRequestEntity true "Brand data"
// @Success 200 {object} entities.BrandEntity "Successfully updated brand"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /brands/{id} [put]
// @Security BearerAuth
func (uc *BrandUseCase) Update(c *gin.Context) {
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

	var request entities.UpsertBrandRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid brand update payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Brand update validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	brand, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Brand not found for update", map[string]interface{}{"brand_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("brand_not_found", "Marca não encontrada"))
			return
		}
		uc.logger.Error(ctx, "Failed to get brand for update", map[string]interface{}{"brand_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Enforce unique brand name within the organization (excluding this brand).
	if request.Name != brand.Name {
		exists, err := uc.repository.Exists(request.Name, organizationID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check brand name existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
			coreErrors.Respond(c, err)
			return
		}
		if exists {
			uc.logger.Warning(ctx, "Brand update failed: name already exists", map[string]interface{}{"name": request.Name})
			coreErrors.Respond(c, coreErrors.Conflict("brand_name_taken", "Já existe uma marca com este nome"))
			return
		}
	}

	brand.Name = request.Name
	brand.Description = request.Description

	if err := uc.repository.Update(brand); err != nil {
		uc.logger.Error(ctx, "Failed to update brand", map[string]interface{}{"brand_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Brand updated successfully", map[string]interface{}{"brand_id": brand.ID})

	c.JSON(http.StatusOK, brand)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityBrand,
		EntityID:       brand.ID.String(),
		EntityName:     brand.Name,
		Description:    "Brand updated: " + brand.Name,
	})
}
