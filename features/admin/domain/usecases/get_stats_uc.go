package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
)

// GetStatsUseCase handles getting platform analytics (admin only)
type GetStatsUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewGetStatsUseCase creates a new instance of GetStatsUseCase
func NewGetStatsUseCase(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *GetStatsUseCase {
	return &GetStatsUseCase{
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// Execute gets platform analytics and stats (PlatformAdmin only)
func (uc *GetStatsUseCase) Execute(ctx context.Context, userRoles []string) (*adminEntities.AdminStats, error) {
	uc.logger.Info(ctx, "Admin getting platform stats", nil)

	// Check if user is PlatformAdmin
	if !contains(userRoles, roles.PlatformAdminRole) {
		uc.logger.Error(ctx, "Non-admin user attempted to get platform stats", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can access platform stats")
	}

	// Get all companies to calculate stats
	companies, _, err := uc.companyRepository.FindAllPaginated(ctx, 1, 1000, "") // Get all companies
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch companies for stats", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.InternalServerError("Failed to fetch platform stats")
	}

	// Calculate stats
	stats := &adminEntities.AdminStats{
		TotalCompanies:       len(companies),
		ActiveSubscriptions:  0,
		TrialSubscriptions:   0,
		OverdueSubscriptions: 0,
		TotalMRR:             0.0,
		ChurnRate:            0.0,
	}

	// Count subscriptions by status
	for _, company := range companies {
		switch company.SubscriptionStatus {
		case "active":
			stats.ActiveSubscriptions++
			// For now, assuming basic MRR calculation - this should be enhanced with actual plan pricing
			if company.CurrentPlan != nil {
				stats.TotalMRR += company.CurrentPlan.Price
			}
		case "trial":
			stats.TrialSubscriptions++
		case "overdue":
			stats.OverdueSubscriptions++
		}
	}

	// Calculate churn rate (simplified calculation)
	// This is a basic calculation - for production, you'd want historical data
	if stats.TotalCompanies > 0 {
		cancelledCount := 0
		for _, company := range companies {
			if company.SubscriptionStatus == "cancelled" {
				cancelledCount++
			}
		}
		stats.ChurnRate = (float64(cancelledCount) / float64(stats.TotalCompanies)) * 100
	}

	uc.logger.Info(ctx, "Platform stats retrieved successfully", map[string]interface{}{
		"total_companies":       stats.TotalCompanies,
		"active_subscriptions":  stats.ActiveSubscriptions,
		"trial_subscriptions":   stats.TrialSubscriptions,
		"overdue_subscriptions": stats.OverdueSubscriptions,
		"total_mrr":             stats.TotalMRR,
		"churn_rate":            stats.ChurnRate,
	})

	return stats, nil
}
