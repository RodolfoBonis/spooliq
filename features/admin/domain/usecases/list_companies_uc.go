package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
)

// ListCompaniesUseCase handles listing all companies (admin only)
type ListCompaniesUseCase struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewListCompaniesUseCase creates a new instance of ListCompaniesUseCase
func NewListCompaniesUseCase(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *ListCompaniesUseCase {
	return &ListCompaniesUseCase{
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// Execute lists all companies with pagination (PlatformAdmin only)
// @Summary List all companies
// @Description Lists all companies in the system with pagination (PlatformAdmin only)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Param status query string false "Filter by subscription status"
// @Success 200 {object} adminEntities.ListCompaniesResponse "List of companies"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/admin/companies [get]
func (uc *ListCompaniesUseCase) Execute(ctx context.Context, userRoles []string, page, pageSize int, statusFilter string) (*adminEntities.ListCompaniesResponse, error) {
	uc.logger.Info(ctx, "Admin listing companies", map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
		"filter":    statusFilter,
	})

	// Check if user is PlatformAdmin
	isPlatformAdmin := contains(userRoles, roles.PlatformAdminRole)
	if !isPlatformAdmin {
		uc.logger.Error(ctx, "Non-admin user attempted to list all companies", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("Only PlatformAdmin can list all companies")
	}

	// Set defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// NOTE: This implementation requires a FindAllPaginated method in CompanyRepository
	// For now, we'll return a placeholder response
	// Full implementation would:
	// 1. Call companyRepository.FindAllPaginated(ctx, page, pageSize, statusFilter)
	// 2. Map entities to CompanyListItem
	// 3. Calculate pagination metadata

	uc.logger.Info(ctx, "Companies listed (placeholder)", map[string]interface{}{
		"page":      page,
		"page_size": pageSize,
	})

	// Placeholder response
	response := &adminEntities.ListCompaniesResponse{
		Companies:  []adminEntities.CompanyListItem{},
		Page:       page,
		PageSize:   pageSize,
		TotalCount: 0,
		TotalPages: 0,
	}

	return response, nil
}
