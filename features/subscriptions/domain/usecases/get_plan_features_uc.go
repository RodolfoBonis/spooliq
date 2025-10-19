package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
)

// GetPlanFeaturesResponse represents the response with all plans
type GetPlanFeaturesResponse struct {
	Plans interface{} `json:"plans"`
}

// GetPlanFeaturesUseCase handles retrieving detailed plan information
type GetPlanFeaturesUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewGetPlanFeaturesUseCase creates a new instance
func NewGetPlanFeaturesUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *GetPlanFeaturesUseCase {
	return &GetPlanFeaturesUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute retrieves all plan details from database
func (uc *GetPlanFeaturesUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	// Get only active plans
	plans, err := uc.planRepository.FindAll(ctx, true)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar planos"))
		return
	}

	response := GetPlanFeaturesResponse{
		Plans: plans,
	}

	c.JSON(http.StatusOK, response)
}
