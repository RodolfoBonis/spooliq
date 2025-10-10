package usecases

import (
	log "github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// FilamentUseCase implements filament business logic operations.
type FilamentUseCase struct {
	repository repositories.FilamentRepository
	validator  *validator.Validate
	logger     log.Logger
}

// IFilamentUseCase defines the contract for filament use case operations.
type IFilamentUseCase interface {
	Create(c *gin.Context)
	Update(c *gin.Context)
	FindByID(c *gin.Context)
	FindAll(c *gin.Context)
	Delete(c *gin.Context)
	Search(c *gin.Context)
}

// NewFilamentUseCase creates a new instance of the filament use case.
func NewFilamentUseCase(repository repositories.FilamentRepository, logger log.Logger) IFilamentUseCase {
	return &FilamentUseCase{
		repository: repository,
		validator:  validator.New(),
		logger:     logger,
	}
}
