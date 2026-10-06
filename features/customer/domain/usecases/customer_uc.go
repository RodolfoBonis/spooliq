package usecases

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/gin-gonic/gin"
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
	repository      repositories.CustomerRepository
	logger          logger.Logger
	activityService activityUc.IActivityService
}

// NewCustomerUseCase creates a new instance of CustomerUseCase
func NewCustomerUseCase(repository repositories.CustomerRepository, logger logger.Logger, activityService activityUc.IActivityService) ICustomerUseCase {
	return &CustomerUseCase{
		repository:      repository,
		logger:          logger,
		activityService: activityService,
	}
}

// customerSortWhitelist maps public sort names to safe SQL column expressions.
var customerSortWhitelist = map[string]string{
	"name":       "lower(name)",
	"email":      "lower(email)",
	"created_at": "created_at",
}

// budgetTotalStatuses are the budget statuses counted toward a customer's
// total spend figure.
var budgetTotalStatuses = []string{"printing", "completed"}
