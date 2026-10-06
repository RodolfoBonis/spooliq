package usecases

import (
	"context"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/google/uuid"
)

// UpdateUserUseCase handles user update logic
type UpdateUserUseCase struct {
	userRepository repositories.UserRepository
	logger         logger.Logger
}

// NewUpdateUserUseCase creates a new instance of UpdateUserUseCase
func NewUpdateUserUseCase(
	userRepository repositories.UserRepository,
	logger logger.Logger,
) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// Execute updates a user
func (uc *UpdateUserUseCase) Execute(ctx context.Context, userID uuid.UUID, organizationID string, currentUserID string, userRoles []string, req *entities.UpdateUserRequest) (*entities.UserEntity, error) {
	uc.logger.Info(ctx, "Updating user", map[string]interface{}{
		"user_id":         userID,
		"organization_id": organizationID,
	})

	// 1. Validate request
	if err := validation.Validate(req); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	// 2. Check permissions
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)

	if !isOwner && !isOrgAdmin {
		uc.logger.Error(ctx, "User does not have permission to update users", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.Forbidden("users_update_forbidden", "Você não tem permissão para atualizar usuários")
	}

	// 3. Fetch the user to be updated
	targetUser, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch target user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, err
	}

	if targetUser == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return nil, errors.NotFoundErr("user_not_found", "Usuário não encontrado")
	}

	// 4. Check hierarchical permissions
	// Owner can update anyone (including self)
	// OrgAdmin can only update 'user' type users (not owner, not other admins, not self)
	if isOrgAdmin && !isOwner {
		if targetUser.UserType != "user" {
			uc.logger.Error(ctx, "OrgAdmin cannot update owner or other admins", map[string]interface{}{
				"target_user_type": targetUser.UserType,
			})
			return nil, errors.Forbidden("org_admin_users_only", "Você só pode atualizar usuários comuns")
		}
		if targetUser.ID.String() == currentUserID {
			uc.logger.Error(ctx, "OrgAdmin cannot update self", nil)
			return nil, errors.Forbidden("org_admin_cannot_update_self", "Você não pode atualizar o seu próprio usuário")
		}
	}

	// 5. Update user data
	if req.Name != nil {
		targetUser.Name = *req.Name
	}
	if req.IsActive != nil {
		targetUser.IsActive = *req.IsActive
	}

	// 6. Save to database
	if err := uc.userRepository.Update(ctx, userID, organizationID, targetUser); err != nil {
		uc.logger.Error(ctx, "Failed to update user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, err
	}

	uc.logger.Info(ctx, "User updated successfully", map[string]interface{}{
		"user_id": userID,
		"email":   targetUser.Email,
	})

	return targetUser, nil
}
