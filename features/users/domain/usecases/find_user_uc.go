package usecases

import (
	"context"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
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
		return nil, errors.Forbidden("user_view_forbidden", "Você não tem permissão para visualizar este usuário")
	}

	// Fetch user from database
	user, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, err
	}

	if user == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return nil, errors.NotFoundErr("user_not_found", "Usuário não encontrado")
	}

	uc.logger.Info(ctx, "User found successfully", map[string]interface{}{
		"user_id": userID,
		"email":   user.Email,
	})

	return user, nil
}
