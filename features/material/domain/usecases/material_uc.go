package usecases

import (
	log "github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/material/domain/repositories"
	"github.com/gin-gonic/gin"
)

// MaterialUseCase implements material business logic operations.
type MaterialUseCase struct {
	repository      repositories.MaterialRepository
	logger          log.Logger
	activityService activityUc.IActivityService
}

// IMaterialUseCase defines the contract for material use case operations.
type IMaterialUseCase interface {
	Create(c *gin.Context)
	Update(c *gin.Context)
	FindByID(c *gin.Context)
	FindAll(c *gin.Context)
	Delete(c *gin.Context)
}

// NewMaterialUseCase creates a new instance of the material use case.
func NewMaterialUseCase(repository repositories.MaterialRepository, logger log.Logger, activityService activityUc.IActivityService) IMaterialUseCase {
	return &MaterialUseCase{
		repository:      repository,
		logger:          logger,
		activityService: activityService,
	}
}

// materialSortWhitelist maps public sort names to safe SQL column expressions.
var materialSortWhitelist = map[string]string{
	"name":       "lower(name)",
	"created_at": "created_at",
}
