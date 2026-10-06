package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a specific material by ID.
// @Summary Get Material by ID
// @Description Get a specific 3D printing material by its ID
// @Tags Materials
// @Accept json
// @Produce json
// @Param id path string true "Material ID" format(uuid)
// @Success 200 {object} entities.FindByIDMaterialResponse "Successfully retrieved material"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /materials/{id} [get]
// @Security BearerAuth
func (uc *MaterialUseCase) FindByID(c *gin.Context) {
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

	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Material not found", map[string]interface{}{"material_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("material_not_found", "Material não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve material", map[string]interface{}{"material_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Material retrieved successfully", map[string]interface{}{"material_id": material.ID})

	c.JSON(http.StatusOK, entities.FindByIDMaterialResponse{Data: *material})
}
