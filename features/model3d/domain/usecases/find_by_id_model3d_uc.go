package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID retrieves a single 3D model by its ID, organization-scoped.
// @Summary Get 3D Model by ID
// @Description Get a specific 3D model by its ID.
// @Tags models3d
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Success 200 {object} entities.Model3DEntity "The 3D model"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d/{id} [get]
// @Security BearerAuth
func (uc *Model3DUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, errInvalidModel3DID())
		return
	}

	model, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve 3D model", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, model)
}
