package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
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
// @Summary Delete user
// @Description Deletes a user (Owner can delete anyone except self, OrgAdmin can delete users only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [delete]
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
		return errors.ForbiddenError("You do not have permission to delete users")
	}

	// 2. Fetch the user to be deleted
	targetUser, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch target user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return errors.InternalServerError("Failed to fetch user")
	}

	if targetUser == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return errors.NotFound("User not found")
	}

	// 3. Check hierarchical permissions
	// Owner cannot delete self
	if isOwner && targetUser.ID.String() == currentUserID {
		uc.logger.Error(ctx, "Owner cannot delete self", nil)
		return errors.ForbiddenError("You cannot delete your own user account")
	}

	// Owner cannot be deleted by anyone (including platform admins - this is org-level deletion)
	if targetUser.UserType == "owner" {
		uc.logger.Error(ctx, "Cannot delete owner user", map[string]interface{}{
			"target_user_id": targetUser.ID,
		})
		return errors.ForbiddenError("Owner user cannot be deleted")
	}

	// OrgAdmin can only delete 'user' type users (not admins)
	if isOrgAdmin && !isOwner {
		if targetUser.UserType != "user" {
			uc.logger.Error(ctx, "OrgAdmin cannot delete admin users", map[string]interface{}{
				"target_user_type": targetUser.UserType,
			})
			return errors.ForbiddenError("You can only delete regular users")
		}
	}

	// 4. Delete user from database (soft delete)
	if err := uc.userRepository.Delete(ctx, userID, organizationID); err != nil {
		uc.logger.Error(ctx, "Failed to delete user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return errors.InternalServerError("Failed to delete user")
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

