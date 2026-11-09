package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating an existing supply.
// @Summary Update Supply
// @Description Update an existing non-printed supply (insumo)
// @Tags Supplies
// @Accept json
// @Produce json
// @Param id path string true "Supply ID" format(uuid)
// @Param request body entities.UpsertSupplyRequestEntity true "Supply data"
// @Success 200 {object} entities.SupplyEntity "Successfully updated supply"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /supplies/{id} [put]
// @Security BearerAuth
func (uc *SupplyUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid supply ID", map[string]interface{}{"supply_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_supply_id", "ID de insumo inválido"))
		return
	}

	var request entities.UpsertSupplyRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid supply update payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Supply update validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	supply, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Supply not found for update", map[string]interface{}{"supply_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("supply_not_found", "Insumo não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to get supply for update", map[string]interface{}{"supply_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Enforce unique supply name within the organization (excluding this supply).
	if request.Name != supply.Name {
		exists, err := uc.repository.Exists(request.Name, organizationID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check supply name existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
			coreErrors.Respond(c, err)
			return
		}
		if exists {
			uc.logger.Warning(ctx, "Supply update failed: name already exists", map[string]interface{}{"name": request.Name})
			coreErrors.Respond(c, coreErrors.Conflict("supply_name_taken", "Já existe um insumo com este nome"))
			return
		}
	}

	// Partial PUTs keep what is stored instead of zeroing the price.
	packageSize, packagePrice := request.ResolvePackageUpdate(supply.PackageSize, supply.PackagePrice)
	supply.Name = request.Name
	supply.Unit = request.Unit
	supply.PackageSize = packageSize
	supply.PackagePrice = packagePrice
	supply.UnitCost = entities.DeriveUnitCostCents(packagePrice, packageSize)
	supply.Description = request.Description
	// is_active keeps its stored value when the client omits it.
	if request.IsActive != nil {
		supply.IsActive = *request.IsActive
	}

	if err := uc.repository.Update(supply); err != nil {
		uc.logger.Error(ctx, "Failed to update supply", map[string]interface{}{"supply_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Supply updated successfully", map[string]interface{}{"supply_id": supply.ID})

	c.JSON(http.StatusOK, supply)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntitySupply,
		EntityID:       supply.ID.String(),
		EntityName:     supply.Name,
		Description:    "Supply updated: " + supply.Name,
	})
}
