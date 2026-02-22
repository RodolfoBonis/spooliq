package usecases

import (
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a single 3D model by its ID.
// @Summary Get 3D Model by ID
// Schemes
// @Description Get a specific 3D model by its ID
// @Tags 3D Models
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Success 200 {object} entities.Model3DEntity "Successfully retrieved 3D model"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /models3d/{id} [get]
// @Security Bearer
func (uc *Model3DUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		uc.logger.Error(ctx, "Invalid model ID", map[string]interface{}{
			"model_id": idParam,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid model ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	model, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("3D model not found")
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve 3D model", map[string]interface{}{
			"model_id": id,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	c.JSON(http.StatusOK, model)
}
