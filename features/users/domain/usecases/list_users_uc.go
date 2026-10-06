package usecases

import (
	"context"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
)

// ListUsersUseCase handles listing users logic
type ListUsersUseCase struct {
	userRepository repositories.UserRepository
	logger         logger.Logger
}

// NewListUsersUseCase creates a new instance of ListUsersUseCase
func NewListUsersUseCase(
	userRepository repositories.UserRepository,
	logger logger.Logger,
) *ListUsersUseCase {
	return &ListUsersUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// Execute lists all users in the organization
func (uc *ListUsersUseCase) Execute(ctx context.Context, organizationID string, userRoles []string, q helpers.ListQuery) ([]*entities.UserEntity, int64, error) {
	uc.logger.Info(ctx, "Listing users", map[string]interface{}{
		"organization_id": organizationID,
	})

	// Check permissions (only Owner or OrgAdmin can list users)
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)

	if !isOwner && !isOrgAdmin {
		uc.logger.Error(ctx, "User does not have permission to list users", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, 0, errors.Forbidden("users_list_forbidden", "Você não tem permissão para listar usuários")
	}

	// Fetch the requested page of users for the organization
	users, total, err := uc.userRepository.FindAll(ctx, organizationID, q)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch users", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, 0, err
	}

	uc.logger.Info(ctx, "Users listed successfully", map[string]interface{}{
		"count": len(users),
		"total": total,
	})

	return users, total, nil
}
