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

// Delete handles deleting a supply by ID.
//
// Deletion is a SOFT delete (see SupplyModel.DeletedAt): the supply is hidden from
// the catalog (list/find) and can no longer be attached to budget items, while its
// row remains so budgets that snapshot it keep pricing/recalculating. Historical
// references therefore never block a delete; the budget_item_supplies FK stays
// ON DELETE RESTRICT purely as a safety net against an accidental hard delete.
// @Summary Delete Supply
// @Description Soft-delete a non-printed supply (insumo) by its ID. Budgets that already reference it keep working from their snapshot.
// @Tags Supplies
// @Param id path string true "Supply ID" format(uuid)
// @Success 204 "Successfully deleted supply"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /supplies/{id} [delete]
// @Security BearerAuth
func (uc *SupplyUseCase) Delete(c *gin.Context) {
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

	supply, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Supply not found for deletion", map[string]interface{}{"supply_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("supply_not_found", "Insumo não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to check supply existence for deletion", map[string]interface{}{"supply_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Soft delete: the row is kept (deleted_at is set) so budgets that snapshot this
	// supply keep working. A supply in use is therefore never blocked here.
	if err := uc.repository.Delete(id); err != nil {
		uc.logger.Error(ctx, "Failed to delete supply", map[string]interface{}{"supply_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Supply deleted successfully", map[string]interface{}{"supply_id": supply.ID})

	c.Status(http.StatusNoContent)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntitySupply,
		EntityID:       supply.ID.String(),
		EntityName:     supply.Name,
		Description:    "Supply deleted: " + supply.Name,
	})
}
