package usecases

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CreateUserUseCase handles user creation logic
type CreateUserUseCase struct {
	userRepository repositories.UserRepository
	keycloakAdmin  services.IKeycloakAdminService
	logger         logger.Logger
	validate       *validator.Validate
}

// NewCreateUserUseCase creates a new instance of CreateUserUseCase
func NewCreateUserUseCase(
	userRepository repositories.UserRepository,
	keycloakAdmin services.IKeycloakAdminService,
	logger logger.Logger,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		userRepository: userRepository,
		keycloakAdmin:  keycloakAdmin,
		logger:         logger,
		validate:       validator.New(),
	}
}

// Execute creates a new user
// @Summary Create a new user
// @Description Creates a new user within the organization (Owner and OrgAdmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body entities.CreateUserRequest true "User creation request"
// @Success 201 {object} entities.UserEntity "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 409 {object} map[string]string "User already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users [post]
func (uc *CreateUserUseCase) Execute(ctx context.Context, organizationID string, userRoles []string, req *entities.CreateUserRequest) (*entities.UserEntity, error) {
	uc.logger.Info(ctx, "Creating user", map[string]interface{}{
		"organization_id": organizationID,
		"email":           req.Email,
		"user_type":       req.UserType,
	})

	// 1. Validate request
	if err := uc.validate.Struct(req); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.BadRequestError("Invalid request data")
	}

	// 2. Check permissions (only Owner or OrgAdmin can create users)
	isOwner := contains(userRoles, roles.OwnerRole)
	isOrgAdmin := contains(userRoles, roles.OrgAdminRole)

	if !isOwner && !isOrgAdmin {
		uc.logger.Error(ctx, "User does not have permission to create users", map[string]interface{}{
			"roles": userRoles,
		})
		return nil, errors.ForbiddenError("You do not have permission to create users")
	}

	// 3. Check if email already exists
	existingUser, err := uc.userRepository.FindByEmail(ctx, req.Email)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check existing user", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.InternalServerError("Failed to check existing user")
	}

	if existingUser != nil {
		uc.logger.Error(ctx, "User with email already exists", map[string]interface{}{
			"email": req.Email,
		})
		return nil, errors.ConflictError("User with this email already exists")
	}

	// 4. Validate user type
	if req.UserType != "admin" && req.UserType != "user" {
		uc.logger.Error(ctx, "Invalid user type", map[string]interface{}{
			"user_type": req.UserType,
		})
		return nil, errors.BadRequestError("Invalid user type. Must be 'admin' or 'user'")
	}

	// 5. Create user in Keycloak
	keycloakReq := services.KeycloakUserRequest{
		Username:      req.Email,
		Email:         req.Email,
		EmailVerified: true,
		Enabled:       true,
		FirstName:     req.Name,
		LastName:      "",
	}
	
	keycloakUserID, appErr := uc.keycloakAdmin.CreateUser(ctx, keycloakReq)
	if appErr != nil {
		uc.logger.Error(ctx, "Failed to create user in Keycloak", map[string]interface{}{
			"error": appErr.Message,
			"email": req.Email,
		})
		return nil, errors.ExternalServiceError("Failed to create user in Keycloak")
	}

	// 6. Set user password in Keycloak
	if err := uc.keycloakAdmin.SetUserPassword(ctx, keycloakUserID, req.Password); err != nil {
		uc.logger.Error(ctx, "Failed to set user password in Keycloak", map[string]interface{}{
			"error":            err.Error(),
			"keycloak_user_id": keycloakUserID,
		})
		// Try to clean up Keycloak user (best effort)
		// Note: In production, consider implementing a cleanup job
		return nil, errors.ExternalServiceError("Failed to set user password")
	}

	// 7. Assign role based on user type
	var roleName string
	if req.UserType == "admin" {
		roleName = roles.OrgAdminRole
	} else {
		roleName = roles.UserRole
	}

	if err := uc.keycloakAdmin.AssignRoleToUser(ctx, keycloakUserID, roleName); err != nil {
		uc.logger.Error(ctx, "Failed to assign role to user in Keycloak", map[string]interface{}{
			"error":            err.Error(),
			"keycloak_user_id": keycloakUserID,
			"role":             roleName,
		})
		// Continue anyway - user can be manually assigned role later
		uc.logger.Info(ctx, "User created but role assignment failed", nil)
	}

	// 8. Add user to organization group in Keycloak
	groupName := fmt.Sprintf("org-%s", organizationID)
	if err := uc.keycloakAdmin.AddUserToGroup(ctx, keycloakUserID, groupName); err != nil {
		uc.logger.Error(ctx, "Failed to add user to organization group", map[string]interface{}{
			"error":            err.Error(),
			"keycloak_user_id": keycloakUserID,
			"group_name":       groupName,
		})
		// Continue anyway - user can be manually added to group later
		uc.logger.Info(ctx, "User created but group assignment failed", nil)
	}

	// 9. Create user in database
	userEntity := &entities.UserEntity{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		KeycloakUserID: keycloakUserID,
		Email:          req.Email,
		Name:           req.Name,
		UserType:       req.UserType,
		IsActive:       true,
	}

	if err := uc.userRepository.Create(ctx, userEntity); err != nil {
		uc.logger.Error(ctx, "Failed to create user in database", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, errors.InternalServerError("Failed to create user")
	}

	uc.logger.Info(ctx, "User created successfully", map[string]interface{}{
		"user_id":          userEntity.ID,
		"email":            userEntity.Email,
		"user_type":        userEntity.UserType,
		"keycloak_user_id": keycloakUserID,
	})

	return userEntity, nil
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

