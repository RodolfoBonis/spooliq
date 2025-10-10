package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving all existing brands
// @Summary Find All Brands
// Schemes
// @Description Find All existing filament brands
// @Tags Brands
// @Accept json
// @Produce json
// @Success 200 {object} entities.FindAllBrandsResponse "Successfully List All Brands"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands [get]
// @Security Bearer
func (uc *BrandUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log brands retrieval attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Brands retrieval attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	brands, err := uc.repository.FindAll(organizationID)

	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to retrieve brands", map[string]interface{}{
			"error": err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful retrieval with automatic trace correlation
	uc.logger.Info(ctx, "Brands retrieved successfully", map[string]interface{}{
		"total_brands": len(brands),
	})

	c.JSON(http.StatusOK, entities.FindAllBrandsResponse{Data: brands})
}
