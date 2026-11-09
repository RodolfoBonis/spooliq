package usecases

import (
	log "github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/repositories"
	"github.com/gin-gonic/gin"
)

// SupplyUseCase implements supply (insumo) business logic operations.
type SupplyUseCase struct {
	repository      repositories.SupplyRepository
	logger          log.Logger
	activityService activityUc.IActivityService
}

// ISupplyUseCase defines the contract for supply use case operations.
type ISupplyUseCase interface {
	Create(c *gin.Context)
	Update(c *gin.Context)
	FindByID(c *gin.Context)
	FindAll(c *gin.Context)
	Delete(c *gin.Context)
}

// NewSupplyUseCase creates a new instance of the supply use case.
func NewSupplyUseCase(repository repositories.SupplyRepository, logger log.Logger, activityService activityUc.IActivityService) ISupplyUseCase {
	return &SupplyUseCase{
		repository:      repository,
		logger:          logger,
		activityService: activityService,
	}
}

// supplySortWhitelist maps public sort names to safe SQL column expressions.
// Only these names are ever honored, which keeps the ORDER BY clause free of raw
// user input.
var supplySortWhitelist = map[string]string{
	"name":       "lower(name)",
	"unit_cost":  "unit_cost",
	"created_at": "created_at",
}
