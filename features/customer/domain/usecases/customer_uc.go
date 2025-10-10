package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	sysRoles "github.com/RodolfoBonis/spooliq/core/roles"
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

// isAdmin checks if the user has admin role
func isAdmin(c *gin.Context) bool {
	rolesInterface, exists := c.Get("user_roles")
	if !exists {
		return false
	}

	roles, ok := rolesInterface.([]string)
	if !ok {
		return false
	}

	for _, role := range roles {
		if role == string(sysRoles.AdminRole) {
			return true
		}
	}

	return false
}

// getUserID retrieves the user ID from context
func getUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}

	userIDStr, ok := userID.(string)
	if !ok {
		return ""
	}

	return userIDStr
}
