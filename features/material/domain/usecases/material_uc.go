package usecases

import (
	log "github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/material/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// MaterialUseCase implements material business logic operations.
type MaterialUseCase struct {
	repository repositories.MaterialRepository
	validator  *validator.Validate
	logger     log.Logger
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
func NewMaterialUseCase(repository repositories.MaterialRepository, logger log.Logger) IMaterialUseCase {
	return &MaterialUseCase{
		repository: repository,
		validator:  validator.New(),
		logger:     logger,
	}
}
