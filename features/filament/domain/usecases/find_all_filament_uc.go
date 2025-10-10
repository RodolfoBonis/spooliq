package usecases

import (
	"net/http"
	"strconv"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
)

// FindAll handles retrieving all filaments accessible by the user
// @Summary Find All Filaments
// @Schemes
// @Description Find all 3D printing filaments (user's own + global catalog)
// @Tags Filaments
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} filamentEntities.FindAllFilamentsResponse "Successfully retrieved filaments"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments [get]
// @Security Bearer
func (uc *FilamentUseCase) FindAll(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	// Log filaments retrieval attempt
	uc.logger.Info(ctx, "Filaments retrieval attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	// Extract user data from context
	userID, _ := c.Get("user_id")
	_, ok := userID.(string)
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
	isAdmin := userRoleStr == roles.OrgAdminRole

	// Parse pagination parameters
	page := 1
	limit := 20

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Fetch filaments
	filaments, total, err := uc.repository.FindAll(ctx, organizationID, limit, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve filaments", map[string]interface{}{
			"error": err.Error(),
		})

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful retrieval
	uc.logger.Info(ctx, "Filaments retrieved successfully", map[string]interface{}{
		"total_filaments": total,
		"returned":        len(filaments),
		"page":            page,
		"limit":           limit,
		"is_admin":        isAdmin,
	})

	// Build response with related data
	responses := make([]filamentEntities.FilamentResponse, len(filaments))
	for i, filament := range filaments {
		responses[i] = filamentEntities.FilamentResponse{
			FilamentEntity: filament,
		}

		// Fetch brand information
		if brandInfo, err := uc.repository.GetBrandInfo(ctx, filament.BrandID); err == nil {
			responses[i].Brand = brandInfo
		}

		// Fetch material information
		if materialInfo, err := uc.repository.GetMaterialInfo(ctx, filament.MaterialID); err == nil {
			responses[i].Material = materialInfo
		}
	}

	totalPages := (total + limit - 1) / limit
	response := filamentEntities.FindAllFilamentsResponse{
		Data:       responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}
