package services

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
)

// UserService defines the business logic interface for user operations
type UserService interface {
	// GetUsers retrieves users with pagination and filtering (admin only)
	GetUsers(ctx context.Context, query entities.UserListQuery, requesterID string) ([]*entities.User, error)

	// GetUserByID retrieves a user by ID (admin or self)
	GetUserByID(ctx context.Context, userID string, requesterID string) (*entities.User, error)

	// GetCurrentUser retrieves the current user's profile
	GetCurrentUser(ctx context.Context, userID string) (*entities.User, error)

	// CreateUser creates a new user (admin only)
	CreateUser(ctx context.Context, request *entities.UserCreateRequest, requesterID string) (*entities.User, error)

	// UpdateUser updates a user (admin or self with restrictions)
	UpdateUser(ctx context.Context, userID string, updates *entities.UserUpdateRequest, requesterID string) (*entities.User, error)

	// DeleteUser deletes a user (admin only)
	DeleteUser(ctx context.Context, userID string, requesterID string) error

	// SetUserEnabled enables/disables a user account (admin only)
	SetUserEnabled(ctx context.Context, userID string, enabled bool, requesterID string) error

	// ResetUserPassword resets a user's password (admin only)
	ResetUserPassword(ctx context.Context, userID string, newPassword string, temporary bool, requesterID string) error

	// ManageUserRole adds or removes roles from a user (admin only)
	AddUserRole(ctx context.Context, userID string, role string, requesterID string) error
	RemoveUserRole(ctx context.Context, userID string, role string, requesterID string) error
}
