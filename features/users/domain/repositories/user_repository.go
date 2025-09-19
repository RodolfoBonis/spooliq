package repositories

import (
	"context"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
)

// UserRepository defines the contract for user data persistence
type UserRepository interface {
	// GetUsers retrieves users with optional filtering and pagination
	GetUsers(ctx context.Context, query entities.UserListQuery) ([]*entities.User, error)

	// GetUserByID retrieves a user by their ID
	GetUserByID(ctx context.Context, userID string) (*entities.User, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*entities.User, error)

	// GetUserByUsername retrieves a user by their username
	GetUserByUsername(ctx context.Context, username string) (*entities.User, error)

	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *entities.UserCreateRequest) (*entities.User, error)

	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, userID string, updates *entities.UserUpdateRequest) (*entities.User, error)

	// DeleteUser deletes a user by their ID
	DeleteUser(ctx context.Context, userID string) error

	// SetUserEnabled enables or disables a user account
	SetUserEnabled(ctx context.Context, userID string, enabled bool) error

	// ResetUserPassword resets a user's password
	ResetUserPassword(ctx context.Context, userID string, newPassword string, temporary bool) error

	// GetUserRoles retrieves the roles assigned to a user
	GetUserRoles(ctx context.Context, userID string) ([]string, error)

	// AddUserRole adds a role to a user
	AddUserRole(ctx context.Context, userID string, role string) error

	// RemoveUserRole removes a role from a user
	RemoveUserRole(ctx context.Context, userID string, role string) error
}