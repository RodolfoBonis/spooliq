package usecases

import (
	"errors"
	"net/http"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update handles updating a 3D model's metadata.
// @Summary Update 3D Model
// Schemes
// @Description Update an existing 3D model's metadata
// @Tags 3D Models
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Param request body entities.UpdateModel3DRequest true "Update data"
// @Success 200 {object} entities.Model3DEntity "Successfully updated 3D model"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /models3d/{id} [put]
// @Security Bearer
func (uc *Model3DUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		uc.logger.Error(ctx, "Invalid model ID", map[string]interface{}{
			"model_id": idParam,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid model ID format")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	var request entities.UpdateModel3DRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	model, err := uc.repository.FindByID(id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("3D model not found")
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(http.StatusNotFound, httpError)
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve 3D model for update", map[string]interface{}{
			"model_id": id,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Apply non-nil fields from request
	if request.Name != nil {
		model.Name = *request.Name
	}
	if request.Description != nil {
		model.Description = *request.Description
	}
	if request.CustomerID != nil {
		parsed, err := uuid.Parse(*request.CustomerID)
		if err != nil {
			appError := coreErrors.UsecaseError("Invalid customer_id format")
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
		model.CustomerID = &parsed
	}
	if request.Tags != nil {
		model.Tags = request.Tags
	}
	if request.Notes != nil {
		model.Notes = request.Notes
	}

	if err := uc.repository.Update(model); err != nil {
		uc.logger.Error(ctx, "Failed to update 3D model", map[string]interface{}{
			"model_id": id,
			"error":    err.Error(),
		})
		httpError := coreErrors.NewHTTPError(http.StatusInternalServerError, "Failed to update 3D model")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(ctx, "3D model updated successfully", map[string]interface{}{
		"model_id":   model.ID,
		"model_name": model.Name,
	})

	c.JSON(http.StatusOK, model)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityModel3D,
		EntityID:       model.ID.String(),
		EntityName:     model.Name,
		Description:    "3D Model updated: " + model.Name,
	})
}
