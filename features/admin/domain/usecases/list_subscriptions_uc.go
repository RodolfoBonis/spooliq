package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
)

// ListSubscriptionsUseCase handles listing all subscriptions (admin only)
type ListSubscriptionsUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewListSubscriptionsUseCase creates a new instance of ListSubscriptionsUseCase
func NewListSubscriptionsUseCase(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *ListSubscriptionsUseCase {
	return &ListSubscriptionsUseCase{
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// Execute lists all subscriptions with pagination (PlatformAdmin only)
// @Summary List all subscriptions
// @Description Lists all subscriptions in the system with pagination (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param status query string false "Filter by subscription status"
// @Success 200 {object} adminEntities.ListSubscriptionsResponse "List of subscriptions"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/subscriptions [get]
func (uc *ListSubscriptionsUseCase) Execute(ctx context.Context, userRoles []string, page, pageSize int, statusFilter string) (*adminEntities.ListSubscriptionsResponse, error) {
	uc.logger.Info(ctx, "Admin listing subscriptions", map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
		"filter":    statusFilter,
	})

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to list all subscriptions", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can list all subscriptions")
	}

	// Set defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Fetch companies (subscriptions are company-based)
	companies, totalCount, err := uc.companyRepository.FindAllPaginated(ctx, page, pageSize, statusFilter)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch subscriptions", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.InternalServerError("Failed to fetch subscriptions")
	}

	// Map entities to subscription list items
	subscriptionItems := make([]adminEntities.SubscriptionListItem, len(companies))
	for i, company := range companies {
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

		subscriptionItems[i] = adminEntities.SubscriptionListItem{
			OrganizationID:     company.OrganizationID,
			CompanyName:        company.Name,
			SubscriptionStatus: company.SubscriptionStatus,
			SubscriptionPlanID: uuidPtrToStrPtr(company.SubscriptionPlanID), // Convert UUID* to string*
			CurrentPlan:        currentPlan,
			TrialEndsAt:        company.TrialEndsAt,
			CreatedAt:          company.CreatedAt,
		}
	}

	// Calculate total pages
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	response := &adminEntities.ListSubscriptionsResponse{
		Subscriptions: subscriptionItems,
		Page:          page,
		PageSize:      pageSize,
		TotalCount:    totalCount,
		TotalPages:    totalPages,
	}

	uc.logger.Info(ctx, "Subscriptions listed successfully", map[string]interface{}{
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})

	return response, nil
}
