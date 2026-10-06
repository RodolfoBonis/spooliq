package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a filament by ID.
// @Summary Find Filament By ID
// @Description Retrieve a single filament by its ID
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Success 200 {object} filamentEntities.FilamentResponse "Successfully retrieved filament"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments/{id} [get]
// @Security BearerAuth
func (uc *FilamentUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{"filament_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_filament_id", "ID de filamento inválido"))
		return
	}

	filament, err := uc.repository.FindByID(ctx, id, organizationID)
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

	uc.logger.Info(ctx, "Filament retrieved successfully", map[string]interface{}{"filament_id": filament.ID})

	response := &filamentEntities.FilamentResponse{FilamentEntity: filament}
	if brandInfo, err := uc.repository.GetBrandInfo(ctx, filament.BrandID); err == nil {
		response.Brand = brandInfo
	}
	if materialInfo, err := uc.repository.GetMaterialInfo(ctx, filament.MaterialID); err == nil {
		response.Material = materialInfo
	}

	c.JSON(http.StatusOK, response)
}
