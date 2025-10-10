package usecases

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/material/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating an existing material.
// @Summary Update Material
// @Schemes
// @Description Update an existing 3D printing material
// @Tags Materials
// @Accept json
// @Produce json
// @Param id path string true "Material ID" format(uuid)
// @Param request body entities.UpsertMaterialRequestEntity true "Material data"
// @Success 200 {object} entities.MaterialEntity "Successfully updated material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /materials/{id} [put]
// @Security Bearer
func (uc *MaterialUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}


	// Log material update attempt (automatic trace correlation via enhanced observability)
	uc.logger.Info(ctx, "Material update attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid material ID", map[string]interface{}{
			"material_id": idParam,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	var request entities.UpsertMaterialRequestEntity

	if err := c.BindJSON(&request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Invalid material update payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Material update validation failed", map[string]interface{}{
			"error":             err.Error(),
			"validation_failed": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	material, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Material not found")
			httpError := appError.ToHTTPError()

			// Enhanced logging with automatic trace correlation
			uc.logger.Error(ctx, "Material not found for update", map[string]interface{}{
				"material_id": id,
				"error":       err.Error(),
			})

			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to get material for update", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if another material with the same name exists (excluding current material)
	if request.Name != material.Name {
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
			uc.logger.Error(ctx, "Material update failed: name already exists", map[string]interface{}{
				"name":     request.Name,
				"conflict": true,
			})

			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Update material fields
	material.Name = request.Name
	material.Description = request.Description
	material.TempTable = request.TempTable
	material.TempExtruder = request.TempExtruder

	// Save updated material
	if err := uc.repository.Update(material); err != nil {
		// Enhanced logging with automatic trace correlation
		uc.logger.Error(ctx, "Failed to update material", map[string]interface{}{
			"material_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError("Failed to update material")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful update with automatic trace correlation
	uc.logger.Info(ctx, "Material updated successfully", map[string]interface{}{
		"material_id":   material.ID,
		"material_name": material.Name,
	})

	c.JSON(200, material)
}
