package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
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
// @Summary List all users
// @Description Lists all users within the organization (Owner and OrgAdmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} entities.UserEntity "List of users"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users [get]
func (uc *ListUsersUseCase) Execute(ctx context.Context, organizationID string, userRoles []string) ([]*entities.UserEntity, error) {
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
		return nil, errors.ForbiddenError("You do not have permission to list users")
	}

	// Fetch all users for the organization
	users, err := uc.userRepository.FindAll(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch users", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.InternalServerError("Failed to fetch users")
	}

	uc.logger.Info(ctx, "Users listed successfully", map[string]interface{}{
		"count": len(users),
	})

	return users, nil
}

