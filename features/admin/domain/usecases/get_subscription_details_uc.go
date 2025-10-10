package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/google/uuid"
)

// GetSubscriptionDetailsUseCase handles getting subscription details (admin only)
type GetSubscriptionDetailsUseCase struct {
	companyRepository      companyRepositories.CompanyRepository
	subscriptionRepository subscriptionRepositories.SubscriptionRepository
	logger                 logger.Logger
}

// NewGetSubscriptionDetailsUseCase creates a new instance of GetSubscriptionDetailsUseCase
func NewGetSubscriptionDetailsUseCase(
	companyRepository companyRepositories.CompanyRepository,
	subscriptionRepository subscriptionRepositories.SubscriptionRepository,
	logger logger.Logger,
) *GetSubscriptionDetailsUseCase {
	return &GetSubscriptionDetailsUseCase{
		companyRepository:      companyRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
	}
}

// Execute gets detailed subscription information (PlatformAdmin only)
// @Summary Get subscription details
// @Description Gets detailed information about a company's subscription (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Success 200 {object} adminEntities.SubscriptionDetailsResponse "Subscription details"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions/{organization_id} [get]
func (uc *GetSubscriptionDetailsUseCase) Execute(ctx context.Context, userRoles []string, organizationID uuid.UUID) (*adminEntities.SubscriptionDetailsResponse, error) {
	uc.logger.Info(ctx, "Admin getting subscription details", map[string]interface{}{
		"organization_id": organizationID,
	})

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to view subscription details", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can view subscription details")
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

	// Fetch recent payments
	// NOTE: This requires FindRecentByOrganizationID method in SubscriptionRepository
	recentPayments := []adminEntities.PaymentHistoryItem{}

	// Map to response
	response := &adminEntities.SubscriptionDetailsResponse{
		OrganizationID:        company.OrganizationID,
		CompanyName:           company.Name,
		SubscriptionStatus:    company.SubscriptionStatus,
		SubscriptionPlan:      company.SubscriptionPlan,
		AsaasCustomerID:       company.AsaasCustomerID,
		AsaasSubscriptionID:   company.AsaasSubscriptionID,
		TrialEndsAt:           company.TrialEndsAt,
		SubscriptionStartedAt: company.SubscriptionStartedAt,
		NextPaymentDue:        company.NextPaymentDue,
		LastPaymentCheck:      company.LastPaymentCheck,
		RecentPayments:        recentPayments,
	}

	uc.logger.Info(ctx, "Subscription details retrieved", map[string]interface{}{
		"organization_id": organizationID,
		"company_name":    company.Name,
	})

	return response, nil
}
