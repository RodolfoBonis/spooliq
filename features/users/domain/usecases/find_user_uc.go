package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/google/uuid"
)

// FindUserUseCase handles finding a user by ID logic
type FindUserUseCase struct {
	userRepository repositories.UserRepository
	logger         logger.Logger
}

// NewFindUserUseCase creates a new instance of FindUserUseCase
func NewFindUserUseCase(
	userRepository repositories.UserRepository,
	logger logger.Logger,
) *FindUserUseCase {
	return &FindUserUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// Execute finds a user by ID
// @Summary Get user by ID
// @Description Gets a user by ID (Owner, OrgAdmin, or self)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} entities.UserEntity "User details"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [get]
func (uc *FindUserUseCase) Execute(ctx context.Context, userID uuid.UUID, organizationID string, currentUserID string, userRoles []string) (*entities.UserEntity, error) {
	uc.logger.Info(ctx, "Finding user by ID", map[string]interface{}{
		"user_id":         userID,
		"organization_id": organizationID,
	})

	// Check permissions
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)
	isSelf := userID.String() == currentUserID

	// Owner and OrgAdmin can view any user, others can only view themselves
	if !isOwner && !isOrgAdmin && !isSelf {
		uc.logger.Error(ctx, "User does not have permission to view this user", map[string]interface{}{
			"roles":           userRoles,
			"requested_user":  userID,
			"current_user_id": currentUserID,
		})
		return nil, errors.ForbiddenError("You do not have permission to view this user")
	}

	// Fetch user from database
	user, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, errors.InternalServerError("Failed to fetch user")
	}

	if user == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return nil, errors.NotFound("User not found")
	}

	uc.logger.Info(ctx, "User found successfully", map[string]interface{}{
		"user_id": userID,
		"email":   user.Email,
	})

	return user, nil
}

