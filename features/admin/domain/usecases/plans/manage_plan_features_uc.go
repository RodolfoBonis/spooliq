package plans

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AddFeatureRequest represents the request to add a feature
type AddFeatureRequest struct {
	PlanID      string `json:"plan_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Description string `json:"description"`
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type" binding:"required,oneof=number boolean text"`
	Available   bool   `json:"available"`
	SortOrder   int    `json:"sort_order"`
}

// UpdateFeatureRequest represents the request to update a feature
type UpdateFeatureRequest struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type" binding:"oneof=number boolean text"`
	Available   *bool  `json:"available"`
	SortOrder   *int   `json:"sort_order"`
}

// AddPlanFeatureUseCase handles adding features to plans
type AddPlanFeatureUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewAddPlanFeatureUseCase creates a new instance
func NewAddPlanFeatureUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *AddPlanFeatureUseCase {
	return &AddPlanFeatureUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute adds a feature to a plan
func (uc *AddPlanFeatureUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	var req AddFeatureRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Dados inválidos: "+err.Error()))
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "ID de plano inválido"))
		return
	}

	// Create feature entity
	feature := &entities.PlanFeature{
		ID:          uuid.New(),
		PlanID:      planID,
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
		Value:       req.Value,
		ValueType:   req.ValueType,
		Available:   req.Available,
		SortOrder:   req.SortOrder,
	}

	// Save to database
	if err := uc.planRepository.AddFeature(ctx, feature); err != nil {
		uc.logger.Error(ctx, "Failed to add feature", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": planID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao adicionar feature"))
		return
	}

	uc.logger.Info(ctx, "Feature added successfully", map[string]interface{}{
		"feature_id": feature.ID,
		"plan_id":    planID,
	})

	c.JSON(http.StatusCreated, feature)
}

// UpdatePlanFeatureUseCase handles updating plan features
type UpdatePlanFeatureUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewUpdatePlanFeatureUseCase creates a new instance
func NewUpdatePlanFeatureUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *UpdatePlanFeatureUseCase {
	return &UpdatePlanFeatureUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute updates a plan feature
func (uc *UpdatePlanFeatureUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	featureIDStr := c.Param("featureId")
	featureID, err := uuid.Parse(featureIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "ID de feature inválido"))
		return
	}

	var req UpdateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Dados inválidos: "+err.Error()))
		return
	}

	// Create feature entity with updates
	feature := &entities.PlanFeature{
		ID:          featureID,
		Name:        req.Name,
		Key:         req.Key,
		Description: req.Description,
		Value:       req.Value,
		ValueType:   req.ValueType,
	}

	if req.Available != nil {
		feature.Available = *req.Available
	}
	if req.SortOrder != nil {
		feature.SortOrder = *req.SortOrder
	}

	// Update in database
	if err := uc.planRepository.UpdateFeature(ctx, feature); err != nil {
		uc.logger.Error(ctx, "Failed to update feature", map[string]interface{}{
			"error":      err.Error(),
			"feature_id": featureID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao atualizar feature"))
		return
	}

	uc.logger.Info(ctx, "Feature updated successfully", map[string]interface{}{
		"feature_id": featureID,
	})

	c.JSON(http.StatusOK, feature)
}

// DeletePlanFeatureUseCase handles deleting plan features
type DeletePlanFeatureUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewDeletePlanFeatureUseCase creates a new instance
func NewDeletePlanFeatureUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *DeletePlanFeatureUseCase {
	return &DeletePlanFeatureUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute deletes a plan feature
func (uc *DeletePlanFeatureUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	featureIDStr := c.Param("featureId")
	featureID, err := uuid.Parse(featureIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "ID de feature inválido"))
		return
	}

	// Delete from database
	if err := uc.planRepository.DeleteFeature(ctx, featureID); err != nil {
		uc.logger.Error(ctx, "Failed to delete feature", map[string]interface{}{
			"error":      err.Error(),
			"feature_id": featureID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao deletar feature"))
		return
	}

	uc.logger.Info(ctx, "Feature deleted successfully", map[string]interface{}{
		"feature_id": featureID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feature deletada com sucesso",
	})
}
