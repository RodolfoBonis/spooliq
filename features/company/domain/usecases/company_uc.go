package usecases

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ICompanyUseCase defines the interface for company use cases
type ICompanyUseCase interface {
	Create(c *gin.Context)
	Get(c *gin.Context)
	Update(c *gin.Context)
	UploadLogo(c *gin.Context)
}

// CompanyUseCase implements the company use cases
type CompanyUseCase struct {
	repository repositories.CompanyRepository
	cdnService *services.CDNService
	validator  *validator.Validate
	logger     logger.Logger
}

// NewCompanyUseCase creates a new instance of CompanyUseCase
func NewCompanyUseCase(
	repository repositories.CompanyRepository,
	cdnService *services.CDNService,
	logger logger.Logger,
) ICompanyUseCase {
	return &CompanyUseCase{
		repository: repository,
		cdnService: cdnService,
		validator:  validator.New(),
		logger:     logger,
	}
}

// getOrganizationID extracts organization ID from context
func getOrganizationID(c *gin.Context) string {
	if orgID, exists := c.Get("organization_id"); exists {
		if id, ok := orgID.(string); ok {
			return id
		}
	}
	return ""
}
