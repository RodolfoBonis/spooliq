package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create handles creating a new material.
// @Summary Create Material
// @Schemes
// @Description Create a new 3D printing material
// @Tags Materials
// @Accept json
// @Produce json
// @Param request body entities.UpsertMaterialRequestEntity true "Material data"
// @Success 201 {object} entities.MaterialEntity "Successfully created material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /materials [post]
// @Security Bearer
func (uc *MaterialUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log material creation attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Material creation attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	var request entities.UpsertMaterialRequestEntity

	if err := c.BindJSON(&request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid material creation payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Material creation validation failed", map[string]interface{}{
			"error":             err.Error(),
			"validation_failed": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if material with the same name already exists
	exists, err := uc.repository.Exists(request.Name, organizationID)
	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to check material name existence", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if exists {
		appError := coreErrors.UsecaseError("Material with this name already exists")
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Material creation failed: name already exists", map[string]interface{}{
			"name":     request.Name,
			"conflict": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Create new material entity
	material := entities.MaterialEntity{
		ID:           uuid.New(),
		Name:         request.Name,
		Description:  request.Description,
		TempTable:    request.TempTable,
		TempExtruder: request.TempExtruder,
	}

	// Save material
	if err := uc.repository.Create(&material); err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to create material", map[string]interface{}{
			"name":  request.Name,
			"error": err.Error(),
		})

		appError := coreErrors.UsecaseError("Failed to create material")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful creation with automatic trace correlation
	uc.logger.Info(ctx, "Material created successfully", map[string]interface{}{
		"material_id":   material.ID,
		"material_name": material.Name,
	})

	c.JSON(http.StatusCreated, material)
}
