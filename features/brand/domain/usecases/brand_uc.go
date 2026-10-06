package usecases

import (
	log "github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/gin-gonic/gin"
)

// BrandUseCase implements brand business logic operations.
type BrandUseCase struct {
	repository      repositories.BrandRepository
	logger          log.Logger
	activityService activityUc.IActivityService
}

// IBrandUseCase defines the contract for brand use case operations.
type IBrandUseCase interface {
	Create(c *gin.Context)
	Update(c *gin.Context)
	FindByID(c *gin.Context)
	FindAll(c *gin.Context)
	Delete(c *gin.Context)
}

// NewBrandUseCase creates a new instance of the brand use case.
func NewBrandUseCase(repository repositories.BrandRepository, logger log.Logger, activityService activityUc.IActivityService) IBrandUseCase {
	return &BrandUseCase{
		repository:      repository,
		logger:          logger,
		activityService: activityService,
	}
}

// brandSortWhitelist maps public sort names to safe SQL column expressions.
// Only these names are ever honored, which keeps the ORDER BY clause free of
// raw user input.
var brandSortWhitelist = map[string]string{
	"name":       "lower(name)",
	"created_at": "created_at",
}
