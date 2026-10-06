package usecases

import (
	"context"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/google/uuid"
)

// DeleteUserUseCase handles user deletion logic
type DeleteUserUseCase struct {
	userRepository repositories.UserRepository
	logger         logger.Logger
}

// NewDeleteUserUseCase creates a new instance of DeleteUserUseCase
func NewDeleteUserUseCase(
	userRepository repositories.UserRepository,
	logger logger.Logger,
) *DeleteUserUseCase {
	return &DeleteUserUseCase{
		userRepository: userRepository,
		logger:         logger,
	}
}

// Execute deletes a user
func (uc *DeleteUserUseCase) Execute(ctx context.Context, userID uuid.UUID, organizationID string, currentUserID string, userRoles []string) error {
	uc.logger.Info(ctx, "Deleting user", map[string]interface{}{
		"user_id":         userID,
		"organization_id": organizationID,
	})

	// 1. Check permissions
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)

	if !isOwner && !isOrgAdmin {
		uc.logger.Error(ctx, "User does not have permission to delete users", map[string]interface{}{
			"roles": userRoles,
		})
		return errors.Forbidden("users_delete_forbidden", "Você não tem permissão para excluir usuários")
	}

	// 2. Fetch the user to be deleted
	targetUser, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch target user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return err
	}

	if targetUser == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return errors.NotFoundErr("user_not_found", "Usuário não encontrado")
	}

	// 3. Check hierarchical permissions
	// Owner cannot delete self
	if isOwner && targetUser.ID.String() == currentUserID {
		uc.logger.Error(ctx, "Owner cannot delete self", nil)
		return errors.Forbidden("owner_cannot_delete_self", "Você não pode excluir a sua própria conta de usuário")
	}

	// Owner cannot be deleted by anyone (including platform admins - this is org-level deletion)
	if targetUser.UserType == "owner" {
		uc.logger.Error(ctx, "Cannot delete owner user", map[string]interface{}{
			"target_user_id": targetUser.ID,
		})
		return errors.Forbidden("owner_cannot_be_deleted", "O usuário proprietário não pode ser excluído")
	}

	// OrgAdmin can only delete 'user' type users (not admins)
	if isOrgAdmin && !isOwner {
		if targetUser.UserType != "user" {
			uc.logger.Error(ctx, "OrgAdmin cannot delete admin users", map[string]interface{}{
				"target_user_type": targetUser.UserType,
			})
			return errors.Forbidden("org_admin_users_only", "Você só pode excluir usuários comuns")
		}
	}

	// 4. Delete user from database (soft delete)
	if err := uc.userRepository.Delete(ctx, userID, organizationID); err != nil {
		uc.logger.Error(ctx, "Failed to delete user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return err
	}

	uc.logger.Info(ctx, "User deleted successfully", map[string]interface{}{
		"user_id": userID,
		"email":   targetUser.Email,
	})

	// Note: Keycloak user deletion is not implemented here for safety
	// Consider implementing a background job or manual cleanup process
	// to deactivate or delete users from Keycloak

	return nil
}
