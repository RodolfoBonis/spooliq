package services

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/users/domain/services"
	"github.com/go-playground/validator/v10"
)

type userServiceImpl struct {
	userRepo  repositories.UserRepository
	logger    logger.Logger
	validator *validator.Validate
}

// NewUserService creates a new user service implementation
func NewUserService(
	userRepo repositories.UserRepository,
	logger logger.Logger,
) services.UserService {
	if userRepo == nil {
		panic("userRepository cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}
	return &userServiceImpl{
		userRepo:  userRepo,
		logger:    logger,
		validator: validator.New(),
	}
}

// GetUsers retrieves users with pagination and filtering (admin only)
func (s *userServiceImpl) GetUsers(ctx context.Context, query entities.UserListQuery, requesterID string) ([]*entities.User, error) {
	if s.userRepo == nil {
		return nil, fmt.Errorf("user repository is not initialized")
	}
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return nil, fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to list users", map[string]interface{}{
			"requester_id": requesterID,
		})
		return nil, entities.ErrUnauthorized
	}

	// Set default pagination if not provided
	if query.Max == 0 {
		query.Max = 20
	}
	if query.Max > 100 {
		query.Max = 100 // Limit maximum results
	}

	users, err := s.userRepo.GetUsers(ctx, query)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get users", err)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	s.logger.Info(ctx, "Users retrieved successfully", map[string]interface{}{
		"requester_id": requesterID,
		"count":        len(users),
		"query":        query.Search,
	})

	return users, nil
}

// GetUserByID retrieves a user by ID (admin or self)
func (s *userServiceImpl) GetUserByID(ctx context.Context, userID string, requesterID string) (*entities.User, error) {
	// Get requester to check permissions
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return nil, fmt.Errorf("failed to verify requester: %w", err)
	}

	// Check if requester can access this user
	if !requester.CanModifyUser(userID) && userID != requesterID {
		s.logger.Warning(ctx, "User attempted to access unauthorized user", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
		})
		return nil, entities.ErrUnauthorized
	}

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get user", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetCurrentUser retrieves the current user's profile
func (s *userServiceImpl) GetCurrentUser(ctx context.Context, userID string) (*entities.User, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get current user", err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	return user, nil
}

// CreateUser creates a new user (admin only)
func (s *userServiceImpl) CreateUser(ctx context.Context, request *entities.UserCreateRequest, requesterID string) (*entities.User, error) {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return nil, fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to create user", map[string]interface{}{
			"requester_id": requesterID,
		})
		return nil, entities.ErrUnauthorized
	}

	// Validate request
	if err := s.validator.Struct(request); err != nil {
		s.logger.LogError(ctx, "User create request validation failed", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := request.ValidateCreate(); err != nil {
		s.logger.LogError(ctx, "User create business validation failed", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetUserByEmail(ctx, request.Email)
	if err == nil && existingUser != nil {
		s.logger.Warning(ctx, "Attempted to create user with existing email", map[string]interface{}{
			"email":        request.Email,
			"requester_id": requesterID,
		})
		return nil, entities.ErrUserAlreadyExists
	}

	existingUser, err = s.userRepo.GetUserByUsername(ctx, request.Username)
	if err == nil && existingUser != nil {
		s.logger.Warning(ctx, "Attempted to create user with existing username", map[string]interface{}{
			"username":     request.Username,
			"requester_id": requesterID,
		})
		return nil, entities.ErrUserAlreadyExists
	}

	// Create user
	user, err := s.userRepo.CreateUser(ctx, request)
	if err != nil {
		s.logger.LogError(ctx, "Failed to create user", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info(ctx, "User created successfully", map[string]interface{}{
		"user_id":      user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"requester_id": requesterID,
	})

	return user, nil
}

// UpdateUser updates a user (admin or self with restrictions)
func (s *userServiceImpl) UpdateUser(ctx context.Context, userID string, updates *entities.UserUpdateRequest, requesterID string) (*entities.User, error) {
	// Get requester to check permissions
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return nil, fmt.Errorf("failed to verify requester: %w", err)
	}

	// Check if requester can modify this user
	if !requester.CanModifyUser(userID) && userID != requesterID {
		s.logger.Warning(ctx, "User attempted to update unauthorized user", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
		})
		return nil, entities.ErrUnauthorized
	}

	// If user is updating themselves, they can't change enabled status
	if userID == requesterID && updates.Enabled != nil {
		s.logger.Warning(ctx, "User attempted to change their own enabled status", map[string]interface{}{
			"user_id": userID,
		})
		updates.Enabled = nil // Remove this field from updates
	}

	// Validate request
	if err := s.validator.Struct(updates); err != nil {
		s.logger.LogError(ctx, "User update request validation failed", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := updates.ValidateUpdate(); err != nil {
		s.logger.LogError(ctx, "User update business validation failed", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if email is being changed and if it already exists
	if updates.Email != nil {
		existingUser, err := s.userRepo.GetUserByEmail(ctx, *updates.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			s.logger.Warning(ctx, "Attempted to update user with existing email", map[string]interface{}{
				"email":        *updates.Email,
				"target_id":    userID,
				"requester_id": requesterID,
			})
			return nil, entities.ErrUserAlreadyExists
		}
	}

	// Update user
	user, err := s.userRepo.UpdateUser(ctx, userID, updates)
	if err != nil {
		s.logger.LogError(ctx, "Failed to update user", err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info(ctx, "User updated successfully", map[string]interface{}{
		"user_id":      userID,
		"requester_id": requesterID,
	})

	return user, nil
}

// DeleteUser deletes a user (admin only)
func (s *userServiceImpl) DeleteUser(ctx context.Context, userID string, requesterID string) error {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to delete user", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
		})
		return entities.ErrUnauthorized
	}

	// Prevent self-deletion
	if userID == requesterID {
		s.logger.Warning(ctx, "Admin attempted to delete themselves", map[string]interface{}{
			"user_id": userID,
		})
		return fmt.Errorf("cannot delete your own account")
	}

	// Verify target user exists
	_, err = s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get target user for deletion", err)
		return fmt.Errorf("failed to get target user: %w", err)
	}

	// Delete user
	err = s.userRepo.DeleteUser(ctx, userID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to delete user", err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	s.logger.Info(ctx, "User deleted successfully", map[string]interface{}{
		"deleted_user_id": userID,
		"requester_id":    requesterID,
	})

	return nil
}

// SetUserEnabled enables/disables a user account (admin only)
func (s *userServiceImpl) SetUserEnabled(ctx context.Context, userID string, enabled bool, requesterID string) error {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to change user enabled status", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
		})
		return entities.ErrUnauthorized
	}

	// Prevent disabling self
	if userID == requesterID && !enabled {
		s.logger.Warning(ctx, "Admin attempted to disable themselves", map[string]interface{}{
			"user_id": userID,
		})
		return fmt.Errorf("cannot disable your own account")
	}

	err = s.userRepo.SetUserEnabled(ctx, userID, enabled)
	if err != nil {
		s.logger.LogError(ctx, "Failed to set user enabled status", err)
		return fmt.Errorf("failed to set user enabled status: %w", err)
	}

	s.logger.Info(ctx, "User enabled status changed", map[string]interface{}{
		"user_id":      userID,
		"enabled":      enabled,
		"requester_id": requesterID,
	})

	return nil
}

// ResetUserPassword resets a user's password (admin only)
func (s *userServiceImpl) ResetUserPassword(ctx context.Context, userID string, newPassword string, temporary bool, requesterID string) error {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to reset password", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
		})
		return entities.ErrUnauthorized
	}

	// Validate password strength
	req := &entities.UserCreateRequest{Password: newPassword}
	if err := req.ValidateCreate(); err != nil {
		return entities.ErrPasswordTooWeak
	}

	err = s.userRepo.ResetUserPassword(ctx, userID, newPassword, temporary)
	if err != nil {
		s.logger.LogError(ctx, "Failed to reset user password", err)
		return fmt.Errorf("failed to reset password: %w", err)
	}

	s.logger.Info(ctx, "User password reset successfully", map[string]interface{}{
		"user_id":      userID,
		"temporary":    temporary,
		"requester_id": requesterID,
	})

	return nil
}

// AddUserRole adds a role to a user (admin only)
func (s *userServiceImpl) AddUserRole(ctx context.Context, userID string, role string, requesterID string) error {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to add role", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
			"role":         role,
		})
		return entities.ErrUnauthorized
	}

	err = s.userRepo.AddUserRole(ctx, userID, role)
	if err != nil {
		s.logger.LogError(ctx, "Failed to add role to user", err)
		return fmt.Errorf("failed to add role: %w", err)
	}

	s.logger.Info(ctx, "Role added to user successfully", map[string]interface{}{
		"user_id":      userID,
		"role":         role,
		"requester_id": requesterID,
	})

	return nil
}

// RemoveUserRole removes a role from a user (admin only)
func (s *userServiceImpl) RemoveUserRole(ctx context.Context, userID string, role string, requesterID string) error {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to remove role", map[string]interface{}{
			"requester_id": requesterID,
			"target_id":    userID,
			"role":         role,
		})
		return entities.ErrUnauthorized
	}

	// Prevent removing admin role from self
	if userID == requesterID && role == "admin" {
		s.logger.Warning(ctx, "Admin attempted to remove admin role from themselves", map[string]interface{}{
			"user_id": userID,
		})
		return fmt.Errorf("cannot remove admin role from your own account")
	}

	err = s.userRepo.RemoveUserRole(ctx, userID, role)
	if err != nil {
		s.logger.LogError(ctx, "Failed to remove role from user", err)
		return fmt.Errorf("failed to remove role: %w", err)
	}

	s.logger.Info(ctx, "Role removed from user successfully", map[string]interface{}{
		"user_id":      userID,
		"role":         role,
		"requester_id": requesterID,
	})

	return nil
}

// GetUserStats retrieves user statistics (admin only)
func (s *userServiceImpl) GetUserStats(ctx context.Context, requesterID string) (*entities.UserStats, error) {
	// Verify requester is admin
	requester, err := s.userRepo.GetUserByID(ctx, requesterID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get requester user", err)
		return nil, fmt.Errorf("failed to verify requester: %w", err)
	}

	if !requester.IsAdmin() {
		s.logger.Warning(ctx, "Non-admin user attempted to get user statistics", map[string]interface{}{
			"requester_id": requesterID,
		})
		return nil, entities.ErrUnauthorized
	}

	// Get all users
	query := entities.UserListQuery{Max: 10000} // Large enough to get all users
	users, err := s.userRepo.GetUsers(ctx, query)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get users for statistics", err)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Calculate statistics
	stats := &entities.UserStats{
		Total:     len(users),
		Active:    0,
		Inactive:  0,
		Suspended: 0,
		Admins:    0,
	}

	for _, user := range users {
		// Count total admins
		if user.IsAdmin() {
			stats.Admins++
		}

		// Count by enabled status
		if user.Enabled {
			stats.Active++
		} else {
			// For Keycloak, disabled users are considered inactive
			// In a more complex scenario, you might want to distinguish between
			// inactive and suspended based on additional attributes
			stats.Inactive++
		}
	}

	s.logger.Info(ctx, "User statistics retrieved successfully", map[string]interface{}{
		"requester_id": requesterID,
		"total":        stats.Total,
		"active":       stats.Active,
		"inactive":     stats.Inactive,
		"admins":       stats.Admins,
	})

	return stats, nil
}
