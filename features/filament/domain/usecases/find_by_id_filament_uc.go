package usecases

import (
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/roles"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FindByID handles retrieving a filament by ID
// @Summary Find Filament By ID
// @Schemes
// @Description Retrieve a single filament by its ID
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Success 200 {object} filamentEntities.FilamentResponse "Successfully retrieved filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [get]
// @Security Bearer
func (uc *FilamentUseCase) FindByID(c *gin.Context) {
	ctx := c.Request.Context()

	// Log filament retrieval attempt
	uc.logger.Info(ctx, "Filament retrieval attempt started", map[string]interface{}{
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

	filament, err := uc.repository.FindByID(ctx, id, userIDStr, isAdmin)
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

		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "forbidden") {
			appError := coreErrors.UsecaseError("Access denied to this filament")
			httpError := appError.ToHTTPError()

			uc.logger.Error(ctx, "Access denied to filament", map[string]interface{}{
				"filament_id": id,
				"user_id":     userIDStr,
			})

			c.AbortWithStatusJSON(http.StatusForbidden, httpError)
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

	// Log successful retrieval
	uc.logger.Info(ctx, "Filament retrieved successfully", map[string]interface{}{
		"filament_id":   filament.ID,
		"filament_name": filament.Name,
	})

	// Build response with related data
	response := &filamentEntities.FilamentResponse{
		FilamentEntity: filament,
	}

	// Fetch brand information
	if brandInfo, err := uc.repository.GetBrandInfo(ctx, filament.BrandID); err == nil {
		response.Brand = brandInfo
	}

	// Fetch material information
	if materialInfo, err := uc.repository.GetMaterialInfo(ctx, filament.MaterialID); err == nil {
		response.Material = materialInfo
	}

	c.JSON(http.StatusOK, response)
}
