package usecases

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	companyRepo "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	customerRepo "github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/gin-gonic/gin"
)

// IBudgetUseCase defines the interface for budget use cases
type IBudgetUseCase interface {
	Create(c *gin.Context)
	Preview(c *gin.Context)
	FindAll(c *gin.Context)
	FindByID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	UpdateStatus(c *gin.Context)
	Duplicate(c *gin.Context)
	Recalculate(c *gin.Context)
	GetCalculation(c *gin.Context)
	FindByCustomer(c *gin.Context)
	GetHistory(c *gin.Context)
	GeneratePDF(c *gin.Context)
	Share(c *gin.Context)
	RevokeShare(c *gin.Context)
}

// BudgetUseCase implements the budget use cases
type BudgetUseCase struct {
	budgetRepository   budgetRepo.BudgetRepository
	customerRepository customerRepo.CustomerRepository
	brandingRepository companyRepo.BrandingRepository
	pdfService         *services.PDFService
	cdnService         *services.CDNService
	logger             logger.Logger
	activityService    activityUc.IActivityService
	// profileProvider / presetProvider back the preset resolver. They are narrow
	// interfaces (defined in this package) implemented by FX-wired adapters over the
	// profile and preset repositories, keeping the dependency one-directional.
	profileProvider ProfilePresetProvider
	presetProvider  DefaultPresetProvider
	// stockDeductor decrements filament stock when a budget is completed. It runs in
	// the same transaction as the status change. Injected via FX (stock feature).
	stockDeductor StockDeductor
}

// NewBudgetUseCase creates a new instance of BudgetUseCase
func NewBudgetUseCase(
	budgetRepository budgetRepo.BudgetRepository,
	customerRepository customerRepo.CustomerRepository,
	brandingRepository companyRepo.BrandingRepository,
	pdfService *services.PDFService,
	cdnService *services.CDNService,
	logger logger.Logger,
	activityService activityUc.IActivityService,
	profileProvider ProfilePresetProvider,
	presetProvider DefaultPresetProvider,
	stockDeductor StockDeductor,
) IBudgetUseCase {
	return &BudgetUseCase{
		budgetRepository:   budgetRepository,
		customerRepository: customerRepository,
		brandingRepository: brandingRepository,
		pdfService:         pdfService,
		cdnService:         cdnService,
		logger:             logger,
		activityService:    activityService,
		profileProvider:    profileProvider,
		presetProvider:     presetProvider,
		stockDeductor:      stockDeductor,
	}
}

// resolvePresets runs the budget preset resolver with this use case's collaborators.
// It is the single entry point the create/preview/update flows use to turn a
// request's optional profile_id + explicit preset IDs into the concrete, validated
// machine/energy/cost preset IDs (and the profile that was used).
func (uc *BudgetUseCase) resolvePresets(c *gin.Context, organizationID string, in PresetResolutionInput) (ResolvedPresets, error) {
	resolver := newPresetResolver(uc.profileProvider, uc.presetProvider, uc.budgetRepository.ValidatePresetInOrg)
	return resolver.Resolve(c.Request.Context(), organizationID, in)
}

// Note: isAdmin and getUserID have been replaced by helpers.GetOrganizationID and helpers.GetUserID
// Organization-wide access control is now handled by organization_id filtering
