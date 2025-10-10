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

// GetPaymentHistoryUseCase handles getting payment history (admin only)
type GetPaymentHistoryUseCase struct {
	companyRepository      companyRepositories.CompanyRepository
	subscriptionRepository subscriptionRepositories.SubscriptionRepository
	logger                 logger.Logger
}

// NewGetPaymentHistoryUseCase creates a new instance of GetPaymentHistoryUseCase
func NewGetPaymentHistoryUseCase(
	companyRepository companyRepositories.CompanyRepository,
	subscriptionRepository subscriptionRepositories.SubscriptionRepository,
	logger logger.Logger,
) *GetPaymentHistoryUseCase {
	return &GetPaymentHistoryUseCase{
		companyRepository:      companyRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
	}
}

// Execute gets payment history for a company (PlatformAdmin only)
// @Summary Get payment history
// @Description Gets payment history for a specific company (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param organization_id path string true "Organization ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} adminEntities.PaymentHistoryResponse "Payment history"
// @Failure 400 {object} map[string]string "Invalid organization ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "Company not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions/{organization_id}/payments [get]
func (uc *GetPaymentHistoryUseCase) Execute(ctx context.Context, userRoles []string, organizationID uuid.UUID, page, pageSize int) (*adminEntities.PaymentHistoryResponse, error) {
	uc.logger.Info(ctx, "Admin getting payment history", map[string]interface{}{
		"organization_id": organizationID,
		"page":            page,
		"page_size":       pageSize,
	})

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to view payment history", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can view payment history")
	}

	// Validate company exists
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

	// Set defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// NOTE: This requires FindByOrganizationIDPaginated method in SubscriptionRepository
	// For now, returning placeholder
	payments := []adminEntities.PaymentHistoryItem{}

	response := &adminEntities.PaymentHistoryResponse{
		Payments:   payments,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: 0,
		TotalPages: 0,
	}

	uc.logger.Info(ctx, "Payment history retrieved (placeholder)", map[string]interface{}{
		"organization_id": organizationID,
	})

	return response, nil
}
