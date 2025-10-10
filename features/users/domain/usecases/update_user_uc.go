package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// UpdateUserUseCase handles user update logic
type UpdateUserUseCase struct {
	userRepository repositories.UserRepository
	logger         logger.Logger
	validate       *validator.Validate
}

// NewUpdateUserUseCase creates a new instance of UpdateUserUseCase
func NewUpdateUserUseCase(
	userRepository repositories.UserRepository,
	logger logger.Logger,
) *UpdateUserUseCase {
	return &UpdateUserUseCase{
		userRepository: userRepository,
		logger:         logger,
		validate:       validator.New(),
	}
}

// Execute updates a user
// @Summary Update user
// @Description Updates a user (Owner can update anyone, OrgAdmin can update users only, not self)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Param request body entities.UpdateUserRequest true "User update request"
// @Success 200 {object} entities.UserEntity "User updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [put]
func (uc *UpdateUserUseCase) Execute(ctx context.Context, userID uuid.UUID, organizationID string, currentUserID string, userRoles []string, req *entities.UpdateUserRequest) (*entities.UserEntity, error) {
	uc.logger.Info(ctx, "Updating user", map[string]interface{}{
		"user_id":         userID,
		"organization_id": organizationID,
	})

	// 1. Validate request
	if err := uc.validate.Struct(req); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.BadRequestError("Invalid request data")
	}

	// 2. Check permissions
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)

	if !isOwner && !isOrgAdmin {
		uc.logger.Error(ctx, "User does not have permission to update users", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("You do not have permission to update users")
	}

	// 3. Fetch the user to be updated
	targetUser, err := uc.userRepository.FindByID(ctx, userID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch target user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, errors.InternalServerError("Failed to fetch user")
	}

	if targetUser == nil {
		uc.logger.Info(ctx, "User not found", map[string]interface{}{
			"user_id": userID,
		})
		return nil, errors.NotFound("User not found")
	}

	// 4. Check hierarchical permissions
	// Owner can update anyone (including self)
	// OrgAdmin can only update 'user' type users (not owner, not other admins, not self)
	if isOrgAdmin && !isOwner {
		if targetUser.UserType != "user" {
			uc.logger.Error(ctx, "OrgAdmin cannot update owner or other admins", map[string]interface{}{
				"target_user_type": targetUser.UserType,
			})
			return nil, errors.ForbiddenError("You can only update regular users")
		}
		if targetUser.ID.String() == currentUserID {
			uc.logger.Error(ctx, "OrgAdmin cannot update self", nil)
			return nil, errors.ForbiddenError("You cannot update your own user")
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
		return nil, errors.InternalServerError("Failed to update user")
	}

	uc.logger.Info(ctx, "User updated successfully", map[string]interface{}{
		"user_id": userID,
		"email":   targetUser.Email,
	})

	return targetUser, nil
}

