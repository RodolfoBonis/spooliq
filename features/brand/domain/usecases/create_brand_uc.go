package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
)

// Create handles creating a new brand.
// @Summary Create Brand
// @Schemes
// @Description Create a new filament brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param request body entities.UpsertBrandRequestEntity true "Brand data"
// @Success 201 {object} entities.BrandEntity "Successfully created brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands [post]
// @Security Bearer
func (uc *BrandUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	// Log brand creation attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Brand creation attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	var request entities.UpsertBrandRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid brand creation payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Brand validation failed", map[string]interface{}{
			"error":             err.Error(),
			"validation_failed": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	exists, err := uc.repository.Exists(request.Name, organizationID)
	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to check brand existence", map[string]interface{}{
			"name":      request.Name,
			"error":     err.Error(),
			"operation": "check_brand_existence",
		})

		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if exists {
		httpError := errors.NewHTTPError(http.StatusConflict, "Brand with this name already exists")

		// Enhanced logging with automatic trace correlation
		uc.logger.Warning(ctx, "Brand creation failed: name already exists", map[string]interface{}{
			"name":     request.Name,
			"conflict": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	brand := &entities.BrandEntity{
		Name:        request.Name,
		Description: request.Description,
	}

	if err := uc.repository.Create(brand); err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to create brand", map[string]interface{}{
			"name":      request.Name,
			"error":     err.Error(),
			"operation": "create_brand",
		})

		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to create brand")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful creation with automatic trace correlation
	uc.logger.Info(ctx, "Brand created successfully", map[string]interface{}{
		"brand_id":   brand.ID,
		"brand_name": brand.Name,
	})

	c.JSON(http.StatusCreated, brand)
}
