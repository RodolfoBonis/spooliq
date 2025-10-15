package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
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
// @Summary Get company details
// @Description Gets detailed information about a specific company (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Success 200 {object} adminEntities.CompanyDetailsResponse "Company details"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies/{organization_id} [get]
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
		SubscriptionPlan:      company.SubscriptionPlan,
		IsPlatformCompany:     company.IsPlatformCompany,
		TrialEndsAt:           company.TrialEndsAt,
		SubscriptionStartedAt: company.SubscriptionStartedAt,
		AsaasCustomerID:       ptrToStr(company.AsaasCustomerID),
		AsaasSubscriptionID:   ptrToStr(company.AsaasSubscriptionID),
		LastPaymentCheck:      company.LastPaymentCheck,
		NextPaymentDue:        company.NextPaymentDue,
		CreatedAt:             company.CreatedAt,
		UpdatedAt:             company.UpdatedAt,
	}

	uc.logger.Info(ctx, "Company details retrieved", map[string]interface{}{
		"organization_id": organizationID,
		"company_name":    company.Name,
	})

	return response, nil
}
