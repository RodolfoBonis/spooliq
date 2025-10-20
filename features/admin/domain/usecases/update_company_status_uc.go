package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// UpdateCompanyStatusUseCase handles updating company subscription status (admin only)
type UpdateCompanyStatusUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
	validate          *validator.Validate
}

// NewUpdateCompanyStatusUseCase creates a new instance of UpdateCompanyStatusUseCase
func NewUpdateCompanyStatusUseCase(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *UpdateCompanyStatusUseCase {
	return &UpdateCompanyStatusUseCase{
		companyRepository: companyRepository,
		logger:            logger,
		validate:          validator.New(),
	}
}

// Execute updates company subscription status (PlatformAdmin only)
// @Summary Update company subscription status
// @Description Manually updates a company's subscription status (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Param request body adminEntities.UpdateStatusRequest true "Status update request"
// @Success 200 {object} adminEntities.CompanyDetailsResponse "Updated company details"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies/{organization_id}/status [patch]
func (uc *UpdateCompanyStatusUseCase) Execute(ctx context.Context, userRoles []string, organizationID uuid.UUID, req *adminEntities.UpdateStatusRequest) (*adminEntities.CompanyDetailsResponse, error) {
	uc.logger.Info(ctx, "Admin updating company status", map[string]interface{}{
		"organization_id": organizationID,
		"new_status":      req.Status,
		"reason":          req.Reason,
	})

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to update company status", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can update company status")
	}

	// Validate request
	if err := uc.validate.Struct(req); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.BadRequestError("Invalid request data")
	}

	// Fetch company from database
	orgIDStr := organizationID.String()
	company, err := uc.companyRepository.FindByOrganizationID(ctx, orgIDStr)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, errors.InternalServerError("Failed to fetch company")
	}

	if company == nil {
		uc.logger.Info(ctx, "Company not found", map[string]interface{}{
			"organization_id": organizationID,
		})
		return nil, errors.NotFound("Company not found")
	}

	// Update status
	oldStatus := company.SubscriptionStatus
	company.SubscriptionStatus = req.Status

	// Save to database
	if err := uc.companyRepository.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company status", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, errors.InternalServerError("Failed to update company status")
	}

	uc.logger.Info(ctx, "Company status updated successfully", map[string]interface{}{
		"organization_id": organizationID,
		"old_status":      oldStatus,
		"new_status":      req.Status,
		"reason":          req.Reason,
	})

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
		StatusUpdatedAt:       &company.StatusUpdatedAt,
		IsPlatformCompany:     company.IsPlatformCompany,
		TrialEndsAt:           company.TrialEndsAt,
		SubscriptionStartedAt: company.SubscriptionStartedAt,
		CreatedAt:             company.CreatedAt,
		UpdatedAt:             company.UpdatedAt,
	}

	return response, nil
}
