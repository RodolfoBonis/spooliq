package usecases

import (
	"errors"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetSliceAnalysis returns the stored slice analysis of a 3D model, recomputing
// the per-slot filament suggestions against the current organization catalog.
// @Summary Get a 3D model's slice analysis
// @Description Returns the stored slicer analysis (print time and per-color filament usage) for a model, enriched with fresh organization filament suggestions.
// @Tags models3d
// @Accept json
// @Produce json
// @Param id path string true "3D Model ID" format(uuid)
// @Success 200 {object} entities.Analysis "Slice analysis with suggestions"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError "Model or slice analysis not found"
// @Failure 500 {object} errors.APIError
// @Router /models3d/{id}/slice-analysis [get]
// @Security BearerAuth
func (uc *Model3DUseCase) GetSliceAnalysis(c *gin.Context) {
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

	model, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve 3D model for slice analysis", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if model.SliceAnalysis == nil {
		coreErrors.Respond(c, errSliceAnalysisNotFound())
		return
	}

	// Recompute suggestions against the current catalog (stored analysis has none).
	analysis := model.SliceAnalysis
	if err := uc.slicerService.Suggest(ctx, organizationID, analysis); err != nil {
		uc.logger.Error(ctx, "Failed to compute filament suggestions", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.JSON(http.StatusOK, analysis)
}
