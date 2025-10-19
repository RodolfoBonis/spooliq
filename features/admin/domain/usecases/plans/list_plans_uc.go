package plans

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
)

// ListPlansUseCase handles listing all plans for admin
type ListPlansUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewListPlansUseCase creates a new instance
func NewListPlansUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *ListPlansUseCase {
	return &ListPlansUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute lists all plans (including inactive)
func (uc *ListPlansUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all plans (not just active ones)
	plans, err := uc.planRepository.FindAll(ctx, false)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve plans", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao buscar planos"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plans": plans,
		"total": len(plans),
	})
}
