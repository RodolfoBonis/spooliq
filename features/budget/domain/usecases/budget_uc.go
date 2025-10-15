package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
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
	GeneratePDF(c *gin.Context)
}

// BudgetUseCase implements the budget use cases
type BudgetUseCase struct {
	budgetRepository   budgetRepo.BudgetRepository
	customerRepository customerRepo.CustomerRepository
	brandingRepository companyRepo.BrandingRepository
	pdfService         *services.PDFService
	cdnService         *services.CDNService
	validator          *validator.Validate
	logger             logger.Logger
}

// NewBudgetUseCase creates a new instance of BudgetUseCase
func NewBudgetUseCase(
	budgetRepository budgetRepo.BudgetRepository,
	customerRepository customerRepo.CustomerRepository,
	brandingRepository companyRepo.BrandingRepository,
	pdfService *services.PDFService,
	cdnService *services.CDNService,
	logger logger.Logger,
) IBudgetUseCase {
	return &BudgetUseCase{
		budgetRepository:   budgetRepository,
		customerRepository: customerRepository,
		brandingRepository: brandingRepository,
		pdfService:         pdfService,
		cdnService:         cdnService,
		validator:          validator.New(),
		logger:             logger,
	}
}

// Note: isAdmin and getUserID have been replaced by helpers.GetOrganizationID and helpers.GetUserID
// Organization-wide access control is now handled by organization_id filtering
