package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving all existing materials
// @Summary Find All Materials
// @Schemes
// @Description Find All existing 3D printing materials
// @Tags Materials
// @Accept json
// @Produce json
// @Success 200 {object} entities.FindAllMaterialsResponse "Successfully List All Materials"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /materials [get]
// @Security Bearer
func (uc *MaterialUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	// Log materials retrieval attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Materials retrieval attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	materials, err := uc.repository.FindAll()

	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to retrieve materials", map[string]interface{}{
			"error": err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful retrieval with automatic trace correlation
	uc.logger.Info(ctx, "Materials retrieved successfully", map[string]interface{}{
		"total_materials": len(materials),
	})

	c.JSON(http.StatusOK, entities.FindAllMaterialsResponse{Data: materials})
}
