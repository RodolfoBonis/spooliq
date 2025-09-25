package usecases

import (
	log "github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BrandUseCase struct {
	repository repositories.BrandRepository
	validator  *validator.Validate
	logger     log.Logger
}

type IBrandUseCase interface {
	Create(c *gin.Context)
	//GetByID(c *gin.Context)
	//GetAll(c *gin.Context)
	//Update(c *gin.Context)
	//Delete(c *gin.Context)
}

func NewBrandUseCase(repository repositories.BrandRepository, logger log.Logger) IBrandUseCase {
	return &BrandUseCase{
		repository: repository,
		validator:  validator.New(),
		logger:     logger,
	}
}
