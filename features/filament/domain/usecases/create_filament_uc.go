package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/roles"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create handles creating a new filament.
// @Summary Create Filament
// @Schemes
// @Description Create a new 3D printing filament
// @Tags Filaments
// @Accept json
// @Produce json
// @Param request body filamentEntities.CreateFilamentRequest true "Filament data"
// @Success 201 {object} filamentEntities.FilamentResponse "Successfully created filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments [post]
// @Security Bearer
func (uc *FilamentUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	// Log filament creation attempt
	uc.logger.Info(ctx, "Filament creation attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	// Extract user data from context
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok {
		appError := errors.UsecaseError("Invalid user ID in context")
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
	isAdmin := userRoleStr == roles.OrgAdmin

	var request filamentEntities.CreateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Invalid filament creation payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		uc.logger.Error(ctx, "Filament validation failed", map[string]interface{}{
			"error":             err.Error(),
			"validation_failed": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate color data if provided
	if request.ColorType != "" && len(request.ColorData) > 0 {
		if !request.ColorType.IsValid() {
			httpError := errors.NewHTTPError(http.StatusBadRequest, "Invalid color type")
			uc.logger.Error(ctx, "Invalid color type", map[string]interface{}{
				"color_type": request.ColorType,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		// Parse and validate color data
		_, err := filamentEntities.ParseColorData(request.ColorType, request.ColorData)
		if err != nil {
			httpError := errors.NewHTTPError(http.StatusBadRequest, "Invalid color data: "+err.Error())
			uc.logger.Error(ctx, "Invalid color data", map[string]interface{}{
				"error":      err.Error(),
				"color_type": request.ColorType,
			})
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Check if filament with same name and brand already exists
	exists, err := uc.repository.ExistsByNameAndBrand(ctx, request.Name, request.BrandID, nil)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check filament existence", map[string]interface{}{
			"name":      request.Name,
			"brand_id":  request.BrandID,
			"error":     err.Error(),
			"operation": "check_filament_existence",
		})

		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if exists {
		httpError := errors.NewHTTPError(http.StatusConflict, "Filament with this name and brand already exists")

		uc.logger.Warning(ctx, "Filament creation failed: already exists", map[string]interface{}{
			"name":     request.Name,
			"brand_id": request.BrandID,
			"conflict": true,
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Create filament entity
	filament := &filamentEntities.FilamentEntity{
		ID:               uuid.New(),
		Name:             request.Name,
		Description:      request.Description,
		BrandID:          request.BrandID,
		MaterialID:       request.MaterialID,
		Color:            request.Color,
		ColorHex:         request.ColorHex,
		ColorType:        request.ColorType,
		ColorData:        request.ColorData,
		Diameter:         request.Diameter,
		Weight:           request.Weight,
		PricePerKg:       request.PricePerKg,
		URL:              request.URL,
		PrintTemperature: request.PrintTemperature,
		BedTemperature:   request.BedTemperature,
	}

	// Set owner_user_id based on the request or user role
	// If request has owner_user_id, use it (only if admin can set it to nil for global)
	if request.OwnerUserID != nil {
		if isAdmin {
			filament.OwnerUserID = request.OwnerUserID
		} else {
			// Non-admin cannot create global filaments
			filament.OwnerUserID = &userIDStr
		}
	} else {
		// Default: set to user's ID
		filament.OwnerUserID = &userIDStr
	}

	// Generate color preview if color data is provided
	if filament.ColorType != "" && len(filament.ColorData) > 0 {
		colorData, err := filamentEntities.ParseColorData(filament.ColorType, filament.ColorData)
		if err == nil {
			filament.ColorPreview = colorData.GenerateCSS()
			// Update legacy color hex for backward compatibility
			filament.ColorHex = filamentEntities.GenerateLegacyColorHex(filament.ColorType, colorData)
		}
	}

	if err := uc.repository.Create(ctx, filament); err != nil {
		uc.logger.Error(ctx, "Failed to create filament", map[string]interface{}{
			"name":      request.Name,
			"error":     err.Error(),
			"operation": "create_filament",
		})

		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to create filament")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(ctx, "Filament created successfully", map[string]interface{}{
		"filament_id":   filament.ID,
		"filament_name": filament.Name,
		"owner_user_id": filament.OwnerUserID,
		"is_global":     filament.OwnerUserID == nil,
	})

	c.JSON(http.StatusCreated, filament)
}
