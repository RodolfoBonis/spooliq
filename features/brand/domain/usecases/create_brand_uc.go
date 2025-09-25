package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
)

func (uc *BrandUseCase) Create(c *gin.Context) {
	var request entities.CreateBrandRequestEntity

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, errors.InvalidRequestResponse(err.Error()))
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationErrorResponse(err.Error()))
		return
	}

	exists, err := uc.repository.Exists(request.Name)

	if err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to check brand existence", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateBrand))
		return
	}

	if exists {
		c.JSON(http.StatusConflict, errors.ErrorResponse("Brand with this name already exists"))
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

		c.JSON(http.StatusInternalServerError, errors.ErrorResponse(errors.ErrorMessages.FailedToCreateBrand))
		return
	}

	c.JSON(http.StatusOK, brand)
}
