package dto

import (
	"time"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
)

// UserResponse represents a user in API responses
type UserResponse struct {
	ID          string            `json:"id"`
	Username    string            `json:"username"`
	Email       string            `json:"email"`
	FirstName   string            `json:"first_name"`
	LastName    string            `json:"last_name"`
	FullName    string            `json:"full_name"`
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Roles       []string          `json:"roles"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// UsersListResponse represents a paginated list of users
type UsersListResponse struct {
	Users []UserResponse `json:"users"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Username          string `json:"username" validate:"required,min=3,max=50"`
	Email            string `json:"email" validate:"required,email"`
	FirstName        string `json:"first_name" validate:"required,min=1,max=100"`
	LastName         string `json:"last_name" validate:"required,min=1,max=100"`
	Password         string `json:"password" validate:"required,min=8"`
	Enabled          bool   `json:"enabled"`
	TemporaryPassword bool   `json:"temporary_password"`
}

// UpdateUserRequest represents the request to update user information
type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

// UserListQueryRequest represents query parameters for listing users
type UserListQueryRequest struct {
	Search string `form:"search" json:"search,omitempty"`
	Page   int    `form:"page" json:"page,omitempty"`
	Size   int    `form:"size" json:"size,omitempty"`
}

// SetUserEnabledRequest represents the request to enable/disable a user
type SetUserEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// ResetPasswordRequest represents the request to reset a user's password
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
	Temporary   bool   `json:"temporary"`
}

// UserRoleRequest represents the request to add/remove a role
type UserRoleRequest struct {
	Role string `json:"role" validate:"required"`
}

// Conversion methods

// ToEntity converts CreateUserRequest to domain entity
func (req *CreateUserRequest) ToEntity() *entities.UserCreateRequest {
	return &entities.UserCreateRequest{
		Username:          req.Username,
		Email:            req.Email,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Password:         req.Password,
		Enabled:          req.Enabled,
		TemporaryPassword: req.TemporaryPassword,
	}
}

// ToEntity converts UpdateUserRequest to domain entity
func (req *UpdateUserRequest) ToEntity() *entities.UserUpdateRequest {
	return &entities.UserUpdateRequest{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Enabled:   req.Enabled,
	}
}

// ToEntity converts UserListQueryRequest to domain entity
func (req *UserListQueryRequest) ToEntity() entities.UserListQuery {
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}
	if req.Size > 100 {
		req.Size = 100
	}

	first := (req.Page - 1) * req.Size

	return entities.UserListQuery{
		Search: req.Search,
		First:  first,
		Max:    req.Size,
	}
}

// FromEntity converts domain entity to UserResponse
func UserResponseFromEntity(user *entities.User) UserResponse {
	response := UserResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		FullName:   user.GetFullName(),
		Enabled:    user.Enabled,
		CreatedAt:  user.CreatedAt,
		UpdatedAt:  user.UpdatedAt,
		Roles:      user.Roles,
		Attributes: user.Attributes,
	}

	return response
}

// FromEntities converts slice of domain entities to UserResponse slice
func UserResponsesFromEntities(users []*entities.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = UserResponseFromEntity(user)
	}
	return responses
}

// ToUsersListResponse creates a paginated response
func ToUsersListResponse(users []*entities.User, page, size int) UsersListResponse {
	return UsersListResponse{
		Users: UserResponsesFromEntities(users),
		Total: len(users),
		Page:  page,
		Size:  size,
	}
}