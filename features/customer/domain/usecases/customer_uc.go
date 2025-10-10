package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ICustomerUseCase defines the interface for customer use cases
type ICustomerUseCase interface {
	Create(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	Search(c *gin.Context)
}

// CustomerUseCase implements the customer use cases
type CustomerUseCase struct {
	repository repositories.CustomerRepository
	validator  *validator.Validate
	logger     logger.Logger
}

// NewCustomerUseCase creates a new instance of CustomerUseCase
func NewCustomerUseCase(repository repositories.CustomerRepository, logger logger.Logger) ICustomerUseCase {
	return &CustomerUseCase{
		repository: repository,
		validator:  validator.New(),
		logger:     logger,
	}
}
