package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
)

// Create handles creating a new brand.
// @Summary Create Brand
// @Description Create a new filament brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param request body entities.UpsertBrandRequestEntity true "Brand data"
// @Success 201 {object} entities.BrandEntity "Successfully created brand"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /brands [post]
// @Security BearerAuth
func (uc *BrandUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	var request entities.UpsertBrandRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid brand creation payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Brand validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	exists, err := uc.repository.Exists(request.Name, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check brand existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if exists {
		uc.logger.Warning(ctx, "Brand creation failed: name already exists", map[string]interface{}{"name": request.Name})
		coreErrors.Respond(c, coreErrors.Conflict("brand_name_taken", "Já existe uma marca com este nome"))
		return
	}

	brand := &entities.BrandEntity{
		Name:           request.Name,
		Description:    request.Description,
		OrganizationID: organizationID,
	}

	if err := uc.repository.Create(brand); err != nil {
		uc.logger.Error(ctx, "Failed to create brand", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Brand created successfully", map[string]interface{}{"brand_id": brand.ID, "brand_name": brand.Name})

	c.JSON(http.StatusCreated, brand)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityBrand,
		EntityID:       brand.ID.String(),
		EntityName:     brand.Name,
		Description:    "Brand created: " + brand.Name,
	})
}
