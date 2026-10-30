package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/gin-gonic/gin"
)

// Create handles creating a new supply.
// @Summary Create Supply
// @Description Create a new non-printed supply (insumo)
// @Tags Supplies
// @Accept json
// @Produce json
// @Param request body entities.UpsertSupplyRequestEntity true "Supply data"
// @Success 201 {object} entities.SupplyEntity "Successfully created supply"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /supplies [post]
// @Security BearerAuth
func (uc *SupplyUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	var request entities.UpsertSupplyRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid supply creation payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Supply validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	exists, err := uc.repository.Exists(request.Name, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check supply existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if exists {
		uc.logger.Warning(ctx, "Supply creation failed: name already exists", map[string]interface{}{"name": request.Name})
		coreErrors.Respond(c, coreErrors.Conflict("supply_name_taken", "Já existe um insumo com este nome"))
		return
	}

	// is_active defaults to true when the client omits it.
	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	// Resolve package pricing (with legacy unit_cost fallback) and keep the stored
	// unit_cost in sync with the derived per-unit value.
	packageSize, packagePrice := request.ResolvePackage()

	supply := &entities.SupplyEntity{
		OrganizationID: organizationID,
		Name:           request.Name,
		Unit:           request.Unit,
		PackageSize:    packageSize,
		PackagePrice:   packagePrice,
		UnitCost:       entities.DeriveUnitCostCents(packagePrice, packageSize),
		Description:    request.Description,
		IsActive:       isActive,
	}

	if err := uc.repository.Create(supply); err != nil {
		uc.logger.Error(ctx, "Failed to create supply", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Supply created successfully", map[string]interface{}{"supply_id": supply.ID, "supply_name": supply.Name})

	c.JSON(http.StatusCreated, supply)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntitySupply,
		EntityID:       supply.ID.String(),
		EntityName:     supply.Name,
		Description:    "Supply created: " + supply.Name,
	})
}
