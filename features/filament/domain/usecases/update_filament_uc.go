package usecases

import (
	"errors"
	"net/http"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating an existing filament
// @Summary Update Filament
// @Schemes
// @Description Update an existing 3D printing filament
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Param request body filamentEntities.UpdateFilamentRequest true "Filament update data"
// @Success 200 {object} filamentEntities.FilamentResponse "Successfully updated filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [put]
// @Security Bearer
func (uc *FilamentUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	// Log filament update attempt
	uc.logger.Info(ctx, "Filament update attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	// Extract user data from context
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok {
		appError := coreErrors.UsecaseError("Invalid user ID in context")
		httpError := appError.ToHTTPError()
		uc.logger.Error(ctx, "Invalid user ID in context", map[string]interface{}{
			"error": "user_id not found or invalid type",
		})
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if user is admin
	userRole, _ := c.Get("user_role")
	userRoleStr, _ := userRole.(string)
	isAdmin := userRoleStr == roles.AdminRole

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{
			"filament_id": idParam,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError("Invalid filament ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	var request filamentEntities.UpdateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Invalid filament update payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Filament validation failed", map[string]interface{}{
			"error":             err.Error(),
			"validation_failed": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate color data if provided
	if request.ColorType != nil && *request.ColorType != "" && request.ColorData != nil && len(*request.ColorData) > 0 {
		if !request.ColorType.IsValid() {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid color type")
			uc.logger.Error(ctx, "Invalid color type", map[string]interface{}{
				"color_type": *request.ColorType,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		// Parse and validate color data
		_, err := filamentEntities.ParseColorData(*request.ColorType, *request.ColorData)
		if err != nil {
			httpError := coreErrors.NewHTTPError(http.StatusBadRequest, "Invalid color data: "+err.Error())
			uc.logger.Error(ctx, "Invalid color data", map[string]interface{}{
				"error":      err.Error(),
				"color_type": *request.ColorType,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Fetch existing filament to check ownership
	existingFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Filament not found")
			httpError := appError.ToHTTPError()

			uc.logger.Error(ctx, "Filament not found", map[string]interface{}{
				"filament_id": id,
				"error":       err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check permissions - only owner or admin can update
	if !isAdmin && (existingFilament.OwnerUserID == nil || *existingFilament.OwnerUserID != userIDStr) {
		appError := coreErrors.UsecaseError("Access denied: you can only update your own filaments")
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Access denied to update filament", map[string]interface{}{
			"filament_id": id,
			"user_id":     userIDStr,
		})

		c.AbortWithStatusJSON(http.StatusForbidden, httpError)
		return
	}

	// Check if new name conflicts with another filament
	if request.Name != nil && *request.Name != existingFilament.Name {
		brandID := existingFilament.BrandID
		if request.BrandID != nil {
			brandID = *request.BrandID
		}

		exists, err := uc.repository.ExistsByNameAndBrand(ctx, *request.Name, brandID, &id)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check filament existence", map[string]interface{}{
				"name":      *request.Name,
				"brand_id":  brandID,
				"error":     err.Error(),
				"operation": "check_filament_existence",
			})

			appError := coreErrors.UsecaseError(err.Error())
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		if exists {
			httpError := coreErrors.NewHTTPError(http.StatusConflict, "Filament with this name and brand already exists")

			uc.logger.Warning(ctx, "Filament update failed: name already exists", map[string]interface{}{
				"name":     *request.Name,
				"brand_id": brandID,
				"conflict": true,
			})

			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Apply updates
	if request.Name != nil {
		existingFilament.Name = *request.Name
	}
	if request.Description != nil {
		existingFilament.Description = *request.Description
	}
	if request.BrandID != nil {
		existingFilament.BrandID = *request.BrandID
	}
	if request.MaterialID != nil {
		existingFilament.MaterialID = *request.MaterialID
	}
	if request.Color != nil {
		existingFilament.Color = *request.Color
	}
	if request.ColorHex != nil {
		existingFilament.ColorHex = *request.ColorHex
	}
	if request.ColorType != nil {
		existingFilament.ColorType = *request.ColorType
	}
	if request.ColorData != nil {
		existingFilament.ColorData = *request.ColorData
	}
	if request.Diameter != nil {
		existingFilament.Diameter = *request.Diameter
	}
	if request.Weight != nil {
		existingFilament.Weight = request.Weight
	}
	if request.PricePerKg != nil {
		existingFilament.PricePerKg = *request.PricePerKg
	}
	if request.URL != nil {
		existingFilament.URL = *request.URL
	}
	if request.PrintTemperature != nil {
		existingFilament.PrintTemperature = request.PrintTemperature
	}
	if request.BedTemperature != nil {
		existingFilament.BedTemperature = request.BedTemperature
	}

	// Regenerate color preview if color data changed
	if request.ColorType != nil && request.ColorData != nil && len(*request.ColorData) > 0 {
		colorData, err := filamentEntities.ParseColorData(*request.ColorType, *request.ColorData)
		if err == nil {
			existingFilament.ColorPreview = colorData.GenerateCSS()
			existingFilament.ColorHex = filamentEntities.GenerateLegacyColorHex(*request.ColorType, colorData)
		}
	}

	// Update in database
	if err := uc.repository.Update(ctx, existingFilament); err != nil {
		uc.logger.Error(ctx, "Failed to update filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
			"operation":   "update_filament",
		})

		httpError := coreErrors.NewHTTPError(http.StatusInternalServerError, "Failed to update filament")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Fetch updated filament with relationships
	updatedFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch updated filament", map[string]interface{}{
			"filament_id": id,
			"error":       err.Error(),
		})

		httpError := coreErrors.NewHTTPError(http.StatusInternalServerError, "Filament updated but failed to fetch details")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(ctx, "Filament updated successfully", map[string]interface{}{
		"filament_id":   id,
		"filament_name": updatedFilament.Name,
	})

	// Build response with related data
	response := &filamentEntities.FilamentResponse{
		FilamentEntity: updatedFilament,
	}

	// Fetch brand information
	if brandInfo, err := uc.repository.GetBrandInfo(ctx, updatedFilament.BrandID); err == nil {
		response.Brand = brandInfo
	}

	// Fetch material information
	if materialInfo, err := uc.repository.GetMaterialInfo(ctx, updatedFilament.MaterialID); err == nil {
		response.Material = materialInfo
	}

	c.JSON(http.StatusOK, response)
}
