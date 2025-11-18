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

	// Calculate offset
	offset := (page - 1) * pageSize

	// Fetch subscription payment history
	subscriptions, err := uc.subscriptionRepository.FindAll(ctx, organizationID, pageSize, offset)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch payment history", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, errors.InternalServerError("Failed to fetch payment history")
	}

	// Get total count
	totalCount, err := uc.subscriptionRepository.CountByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to count payments", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		return nil, errors.InternalServerError("Failed to count payments")
	}

	// Map entities to response items
	paymentItems := make([]adminEntities.PaymentHistoryItem, len(subscriptions))
	for i, subscription := range subscriptions {
		paymentItems[i] = adminEntities.PaymentHistoryItem{
			ID:             subscription.ID.String(),
			Amount:         subscription.Amount,
			Status:         subscription.Status,
			DueDate:        subscription.DueDate,
			PaymentDate:    subscription.PaymentDate,
			InvoiceURL:     subscription.InvoiceURL,
			AsaasPaymentID: subscription.AsaasPaymentID,
			CreatedAt:      subscription.CreatedAt,
		}
	}

	// Calculate total pages
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	response := &adminEntities.PaymentHistoryResponse{
		Payments:   paymentItems,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}

	uc.logger.Info(ctx, "Payment history retrieved successfully", map[string]interface{}{
		"organization_id": organizationID,
		"total_count":     totalCount,
	})

	return response, nil
}
