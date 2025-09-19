package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/logger"
	userEntities "github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
)

type keycloakUserRepository struct {
	client             *gocloak.GoCloak
	keycloakConfig     entities.KeyCloakDataEntity
	logger             logger.Logger
	adminToken         *gocloak.JWT
	adminTokenExpiry   time.Time
}

// NewKeycloakUserRepository creates a new Keycloak user repository
func NewKeycloakUserRepository(
	client *gocloak.GoCloak,
	keycloakConfig entities.KeyCloakDataEntity,
	logger logger.Logger,
) repositories.UserRepository {
	return &keycloakUserRepository{
		client:         client,
		keycloakConfig: keycloakConfig,
		logger:         logger,
	}
}

// GetUsers retrieves users with optional filtering and pagination
func (r *keycloakUserRepository) GetUsers(ctx context.Context, query userEntities.UserListQuery) ([]*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	params := gocloak.GetUsersParams{
		Search: &query.Search,
		First:  &query.First,
		Max:    &query.Max,
	}

	users, err := r.client.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, params)
	if err != nil {
		r.logger.LogError(ctx, "Failed to get users from Keycloak", err)
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var result []*userEntities.User
	for _, kcUser := range users {
		user := r.mapKeycloakUserToEntity(kcUser)

		// Get user roles
		roles, err := r.getUserRoles(ctx, *kcUser.ID)
		if err != nil {
			r.logger.Warning(ctx, "Failed to get user roles", map[string]interface{}{
				"user_id": *kcUser.ID,
				"error":   err.Error(),
			})
		} else {
			user.Roles = roles
		}

		result = append(result, user)
	}

	return result, nil
}

// GetUserByID retrieves a user by their ID
func (r *keycloakUserRepository) GetUserByID(ctx context.Context, userID string) (*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	kcUser, err := r.client.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
	if err != nil {
		r.logger.LogError(ctx, "Failed to get user by ID from Keycloak", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user := r.mapKeycloakUserToEntity(kcUser)

	// Get user roles
	roles, err := r.getUserRoles(ctx, userID)
	if err != nil {
		r.logger.Warning(ctx, "Failed to get user roles", map[string]interface{}{
			"user_id": userID,
			"error":   err.Error(),
		})
	} else {
		user.Roles = roles
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email address
func (r *keycloakUserRepository) GetUserByEmail(ctx context.Context, email string) (*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	params := gocloak.GetUsersParams{
		Email: &email,
		Exact: gocloak.BoolP(true),
	}

	users, err := r.client.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, params)
	if err != nil {
		r.logger.LogError(ctx, "Failed to get user by email from Keycloak", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(users) == 0 {
		return nil, userEntities.ErrUserNotFound
	}

	user := r.mapKeycloakUserToEntity(users[0])

	// Get user roles
	roles, err := r.getUserRoles(ctx, *users[0].ID)
	if err != nil {
		r.logger.Warning(ctx, "Failed to get user roles", map[string]interface{}{
			"user_id": *users[0].ID,
			"error":   err.Error(),
		})
	} else {
		user.Roles = roles
	}

	return user, nil
}

// GetUserByUsername retrieves a user by their username
func (r *keycloakUserRepository) GetUserByUsername(ctx context.Context, username string) (*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	params := gocloak.GetUsersParams{
		Username: &username,
		Exact:    gocloak.BoolP(true),
	}

	users, err := r.client.GetUsers(ctx, token.AccessToken, r.keycloakConfig.Realm, params)
	if err != nil {
		r.logger.LogError(ctx, "Failed to get user by username from Keycloak", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(users) == 0 {
		return nil, userEntities.ErrUserNotFound
	}

	user := r.mapKeycloakUserToEntity(users[0])

	// Get user roles
	roles, err := r.getUserRoles(ctx, *users[0].ID)
	if err != nil {
		r.logger.Warning(ctx, "Failed to get user roles", map[string]interface{}{
			"user_id": *users[0].ID,
			"error":   err.Error(),
		})
	} else {
		user.Roles = roles
	}

	return user, nil
}

// CreateUser creates a new user
func (r *keycloakUserRepository) CreateUser(ctx context.Context, request *userEntities.UserCreateRequest) (*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	// Create user in Keycloak
	kcUser := gocloak.User{
		Username:  &request.Username,
		Email:     &request.Email,
		FirstName: &request.FirstName,
		LastName:  &request.LastName,
		Enabled:   &request.Enabled,
	}

	userID, err := r.client.CreateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, kcUser)
	if err != nil {
		r.logger.LogError(ctx, "Failed to create user in Keycloak", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Set password
	err = r.client.SetPassword(ctx, token.AccessToken, r.keycloakConfig.Realm, userID, request.Password, request.TemporaryPassword)
	if err != nil {
		r.logger.LogError(ctx, "Failed to set password for new user", err)
		// Try to cleanup created user
		_ = r.client.DeleteUser(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
		return nil, fmt.Errorf("failed to set password: %w", err)
	}

	// Get the created user
	return r.GetUserByID(ctx, userID)
}

// UpdateUser updates an existing user
func (r *keycloakUserRepository) UpdateUser(ctx context.Context, userID string, updates *userEntities.UserUpdateRequest) (*userEntities.User, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get current user
	currentUser, err := r.client.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Apply updates
	if updates.Email != nil {
		currentUser.Email = updates.Email
	}
	if updates.FirstName != nil {
		currentUser.FirstName = updates.FirstName
	}
	if updates.LastName != nil {
		currentUser.LastName = updates.LastName
	}
	if updates.Enabled != nil {
		currentUser.Enabled = updates.Enabled
	}

	// Update user in Keycloak
	err = r.client.UpdateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, *currentUser)
	if err != nil {
		r.logger.LogError(ctx, "Failed to update user in Keycloak", err)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return updated user
	return r.GetUserByID(ctx, userID)
}

// DeleteUser deletes a user by their ID
func (r *keycloakUserRepository) DeleteUser(ctx context.Context, userID string) error {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	err = r.client.DeleteUser(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
	if err != nil {
		r.logger.LogError(ctx, "Failed to delete user in Keycloak", err)
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// SetUserEnabled enables or disables a user account
func (r *keycloakUserRepository) SetUserEnabled(ctx context.Context, userID string, enabled bool) error {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get current user
	currentUser, err := r.client.GetUserByID(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Update enabled status
	currentUser.Enabled = &enabled

	err = r.client.UpdateUser(ctx, token.AccessToken, r.keycloakConfig.Realm, *currentUser)
	if err != nil {
		r.logger.LogError(ctx, "Failed to update user enabled status in Keycloak", err)
		return fmt.Errorf("failed to update user enabled status: %w", err)
	}

	return nil
}

// ResetUserPassword resets a user's password
func (r *keycloakUserRepository) ResetUserPassword(ctx context.Context, userID string, newPassword string, temporary bool) error {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	err = r.client.SetPassword(ctx, token.AccessToken, r.keycloakConfig.Realm, userID, newPassword, temporary)
	if err != nil {
		r.logger.LogError(ctx, "Failed to reset user password in Keycloak", err)
		return fmt.Errorf("failed to reset password: %w", err)
	}

	return nil
}

// GetUserRoles retrieves the roles assigned to a user
func (r *keycloakUserRepository) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return r.getUserRoles(ctx, userID)
}

// AddUserRole adds a role to a user
func (r *keycloakUserRepository) AddUserRole(ctx context.Context, userID string, role string) error {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get role by name
	kcRole, err := r.client.GetRealmRole(ctx, token.AccessToken, r.keycloakConfig.Realm, role)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Add role to user
	err = r.client.AddRealmRoleToUser(ctx, token.AccessToken, r.keycloakConfig.Realm, userID, []gocloak.Role{*kcRole})
	if err != nil {
		r.logger.LogError(ctx, "Failed to add role to user in Keycloak", err)
		return fmt.Errorf("failed to add role: %w", err)
	}

	return nil
}

// RemoveUserRole removes a role from a user
func (r *keycloakUserRepository) RemoveUserRole(ctx context.Context, userID string, role string) error {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admin token: %w", err)
	}

	// Get role by name
	kcRole, err := r.client.GetRealmRole(ctx, token.AccessToken, r.keycloakConfig.Realm, role)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Remove role from user
	err = r.client.DeleteRealmRoleFromUser(ctx, token.AccessToken, r.keycloakConfig.Realm, userID, []gocloak.Role{*kcRole})
	if err != nil {
		r.logger.LogError(ctx, "Failed to remove role from user in Keycloak", err)
		return fmt.Errorf("failed to remove role: %w", err)
	}

	return nil
}

// Helper methods

func (r *keycloakUserRepository) getAdminToken(ctx context.Context) (*gocloak.JWT, error) {
	// Check if we have a valid admin token
	if r.adminToken != nil && time.Now().Before(r.adminTokenExpiry.Add(-30*time.Second)) {
		return r.adminToken, nil
	}

	// Get new admin token
	token, err := r.client.LoginAdmin(ctx, r.keycloakConfig.ClientID, r.keycloakConfig.ClientSecret, r.keycloakConfig.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed to login as admin: %w", err)
	}

	r.adminToken = token
	r.adminTokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return token, nil
}

func (r *keycloakUserRepository) getUserRoles(ctx context.Context, userID string) ([]string, error) {
	token, err := r.getAdminToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin token: %w", err)
	}

	roles, err := r.client.GetRealmRolesByUserID(ctx, token.AccessToken, r.keycloakConfig.Realm, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	var roleNames []string
	for _, role := range roles {
		if role.Name != nil {
			roleNames = append(roleNames, *role.Name)
		}
	}

	return roleNames, nil
}

func (r *keycloakUserRepository) mapKeycloakUserToEntity(kcUser *gocloak.User) *userEntities.User {
	user := &userEntities.User{
		ID:        *kcUser.ID,
		Enabled:   *kcUser.Enabled,
		CreatedAt: time.Unix(*kcUser.CreatedTimestamp/1000, 0),
	}

	if kcUser.Username != nil {
		user.Username = *kcUser.Username
	}
	if kcUser.Email != nil {
		user.Email = *kcUser.Email
	}
	if kcUser.FirstName != nil {
		user.FirstName = *kcUser.FirstName
	}
	if kcUser.LastName != nil {
		user.LastName = *kcUser.LastName
	}

	// Set UpdatedAt to CreatedAt if not available
	user.UpdatedAt = user.CreatedAt

	// Convert attributes if present
	if kcUser.Attributes != nil {
		user.Attributes = make(map[string]string)
		for key, values := range *kcUser.Attributes {
			if len(values) > 0 {
				user.Attributes[key] = values[0]
			}
		}
	}

	return user
}