package usecases

import (
	log "github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BrandUseCase implements brand business logic operations.
type BrandUseCase struct {
	repository repositories.BrandRepository
	validator  *validator.Validate
	logger     log.Logger
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
func NewBrandUseCase(repository repositories.BrandRepository, logger log.Logger) IBrandUseCase {
	return &BrandUseCase{
		repository: repository,
		validator:  validator.New(),
		logger:     logger,
	}
}
