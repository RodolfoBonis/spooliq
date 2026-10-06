package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Update updates a 3D model's metadata, organization-scoped.
// @Summary Update 3D Model
// @Description Update an existing 3D model's metadata. Send customer_id: null to detach the customer.
// @Tags models3d
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Param request body entities.UpdateModel3DRequest true "Update data"
// @Success 200 {object} entities.Model3DEntity "The updated 3D model"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d/{id} [put]
// @Security BearerAuth
func (uc *Model3DUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, errInvalidModel3DID())
		return
	}

	var request entities.UpdateModel3DRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		coreErrors.Respond(c, err)
		return
	}
	if err := validation.Validate(request); err != nil {
		coreErrors.Respond(c, err)
		return
	}

	model, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve 3D model for update", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if request.Name != nil {
		model.Name = *request.Name
	}
	if request.Description != nil {
		model.Description = *request.Description
	}
	if request.Tags != nil {
		model.Tags = request.Tags
	}
	if request.Notes != nil {
		model.Notes = request.Notes
	}

	// customer_id has three states: absent (keep), explicit null (detach), value
	// (set + validate ownership). The repository Selects customer_id so an explicit
	// nil is persisted as NULL.
	if request.CustomerID.Set {
		if request.CustomerID.Value == nil {
			model.CustomerID = nil
		} else {
			parsed, err := uuid.Parse(*request.CustomerID.Value)
			if err != nil {
				coreErrors.Respond(c, errInvalidCustomerID())
				return
			}
			exists, err := uc.repository.CustomerExists(ctx, parsed, organizationID)
			if err != nil {
				uc.logger.Error(ctx, "Failed to validate customer", map[string]interface{}{"error": err.Error()})
				coreErrors.Respond(c, err)
				return
			}
			if !exists {
				coreErrors.Respond(c, errCustomerNotFound())
				return
			}
			model.CustomerID = &parsed
		}
	}

	if err := uc.repository.Update(ctx, model); err != nil {
		uc.logger.Error(ctx, "Failed to update 3D model", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	uc.logger.Info(ctx, "3D model updated successfully", map[string]interface{}{"model_id": model.ID, "model_name": model.Name})

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
