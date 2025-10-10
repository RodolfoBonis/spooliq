package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	sysRoles "github.com/RodolfoBonis/spooliq/core/roles"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	customerRepo "github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// IBudgetUseCase defines the interface for budget use cases
type IBudgetUseCase interface {
	Create(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	UpdateStatus(c *gin.Context)
	Duplicate(c *gin.Context)
	Recalculate(c *gin.Context)
	FindByCustomer(c *gin.Context)
	GetHistory(c *gin.Context)
}

// BudgetUseCase implements the budget use cases
type BudgetUseCase struct {
	budgetRepository   budgetRepo.BudgetRepository
	customerRepository customerRepo.CustomerRepository
	validator          *validator.Validate
	logger             logger.Logger
}

// NewBudgetUseCase creates a new instance of BudgetUseCase
func NewBudgetUseCase(
	budgetRepository budgetRepo.BudgetRepository,
	customerRepository customerRepo.CustomerRepository,
	logger logger.Logger,
) IBudgetUseCase {
	return &BudgetUseCase{
		budgetRepository:   budgetRepository,
		customerRepository: customerRepository,
		validator:          validator.New(),
		logger:             logger,
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
