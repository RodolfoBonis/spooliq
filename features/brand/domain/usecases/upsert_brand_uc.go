package usecases

import (
	"net/http"

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
	var request entities.UpsertBrandRequestEntity
	ctx := c.Request.Context()
	if err := c.ShouldBindJSON(&request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	exists, err := uc.repository.Exists(request.Name)

	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to check brand existence", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if exists {
		httpError := errors.NewHTTPError(http.StatusConflict, "Brand with this name already exists")
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	brand := &entities.BrandEntity{
		Name:        request.Name,
		Description: request.Description,
	}

	if err := uc.repository.Create(brand); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to create brand", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})

		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to create brand")
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	c.JSON(http.StatusOK, brand)
}
