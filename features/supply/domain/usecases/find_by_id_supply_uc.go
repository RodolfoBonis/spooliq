package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a specific supply by ID.
// @Summary Get Supply by ID
// @Description Get a specific supply (insumo) by its ID
// @Tags Supplies
// @Accept json
// @Produce json
// @Param id path string true "Supply ID" format(uuid)
// @Success 200 {object} entities.FindByIDSupplyResponse "Successfully retrieved supply"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /supplies/{id} [get]
// @Security BearerAuth
func (uc *SupplyUseCase) FindByID(c *gin.Context) {
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
			uc.logger.Error(ctx, "Supply not found", map[string]interface{}{"supply_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("supply_not_found", "Insumo não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve supply", map[string]interface{}{"supply_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Supply retrieved successfully", map[string]interface{}{"supply_id": supply.ID})

	c.JSON(http.StatusOK, entities.FindByIDSupplyResponse{Data: *supply})
}
