package plans

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeletePlanUseCase handles plan deletion by admin
type DeletePlanUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewDeletePlanUseCase creates a new instance
func NewDeletePlanUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *DeletePlanUseCase {
	return &DeletePlanUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute deletes a plan
func (uc *DeletePlanUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	// Get plan ID from URL
	planIDStr := c.Param("id")
	planID, err := uuid.Parse(planIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "ID de plano inválido"))
		return
	}

	// Delete from database
	if err := uc.planRepository.Delete(ctx, planID); err != nil {
		uc.logger.Error(ctx, "Failed to delete plan", map[string]interface{}{
			"error":   err.Error(),
			"plan_id": planID,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao deletar plano"))
		return
	}

	uc.logger.Info(ctx, "Plan deleted successfully", map[string]interface{}{
		"plan_id": planID,
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Plano deletado com sucesso",
	})
}
