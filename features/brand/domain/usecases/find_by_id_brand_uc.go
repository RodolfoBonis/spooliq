package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a specific brand by ID.
// @Summary Get Brand by ID
// @Description Get a specific filament brand by its ID
// @Tags Brands
// @Accept json
// @Produce json
// @Param id path string true "Brand ID" format(uuid)
// @Success 200 {object} entities.FindByIDBrandResponse "Successfully retrieved brand"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /brands/{id} [get]
// @Security BearerAuth
func (uc *BrandUseCase) FindByID(c *gin.Context) {
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

	brand, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Brand not found", map[string]interface{}{"brand_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("brand_not_found", "Marca não encontrada"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve brand", map[string]interface{}{"brand_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Brand retrieved successfully", map[string]interface{}{"brand_id": brand.ID})

	c.JSON(http.StatusOK, entities.FindByIDBrandResponse{Data: *brand})
}
