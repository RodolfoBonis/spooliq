package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// GetCompanyDetailsUseCase handles getting company details (admin only)
type GetCompanyDetailsUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewGetCompanyDetailsUseCase creates a new instance of GetCompanyDetailsUseCase
func NewGetCompanyDetailsUseCase(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *GetCompanyDetailsUseCase {
	return &GetCompanyDetailsUseCase{
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// Execute gets detailed company information (PlatformAdmin only)
func (uc *GetCompanyDetailsUseCase) Execute(ctx context.Context, userRoles []string, organizationID uuid.UUID) (*adminEntities.CompanyDetailsResponse, error) {
	uc.logger.Info(ctx, "Admin getting company details", map[string]interface{}{
		"organization_id": organizationID,
	})

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to view company details", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can view company details")
	}

	// Convert UUID to string for repository call
	orgIDStr := organizationID.String()

	// Fetch company from database
	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgIDStr)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, errors.InternalServerError("Failed to fetch company details")
	}

	if company == nil {
		uc.logger.Info(ctx, "Company not found", map[string]interface{}{
			"organization_id": organizationID,
		})
		return nil, errors.NotFound("Company not found")
	}

	// Convert plan entity to response
	var currentPlan *subscriptionEntities.SubscriptionPlanResponse
	if company.CurrentPlan != nil {
		currentPlan = &subscriptionEntities.SubscriptionPlanResponse{
			ID:          company.CurrentPlan.ID,
			Name:        company.CurrentPlan.Name,
			Description: company.CurrentPlan.Description,
			Price:       company.CurrentPlan.Price,
			Cycle:       company.CurrentPlan.Cycle,
			Features:    company.CurrentPlan.Features,
			IsActive:    company.CurrentPlan.IsActive,
			CreatedAt:   company.CurrentPlan.CreatedAt,
			UpdatedAt:   company.CurrentPlan.UpdatedAt,
		}
	}

	// Map to response
	response := &adminEntities.CompanyDetailsResponse{
		ID:                    company.ID.String(),
		OrganizationID:        company.OrganizationID,
		Name:                  company.Name,
		Email:                 ptrToStr(company.Email),
		Phone:                 ptrToStr(company.Phone),
		WhatsApp:              ptrToStr(company.WhatsApp),
		Instagram:             ptrToStr(company.Instagram),
		Website:               ptrToStr(company.Website),
		LogoURL:               ptrToStr(company.LogoURL),
		SubscriptionStatus:    company.SubscriptionStatus,
		SubscriptionPlanID:    uuidPtrToStrPtr(company.SubscriptionPlanID), // Convert UUID* to string*
		CurrentPlan:           currentPlan,
		StatusUpdatedAt:       &company.StatusUpdatedAt,
		IsPlatformCompany:     company.IsPlatformCompany,
		TrialEndsAt:           company.TrialEndsAt,
		SubscriptionStartedAt: company.SubscriptionStartedAt,
		CreatedAt:             company.CreatedAt,
		UpdatedAt:             company.UpdatedAt,
	}

	uc.logger.Info(ctx, "Company details retrieved", map[string]interface{}{
		"organization_id": organizationID,
		"company_name":    company.Name,
	})

	return response, nil
}
