package entities

import (
	"time"

	"github.com/google/uuid"
)

// UserEntity represents a user in the domain layer
type UserEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"`
	KeycloakUserID string     `json:"keycloak_user_id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	UserType       string     `json:"user_type"` // 'owner', 'admin', 'user'
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// CreateUserRequest represents the request to create a new user
type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required"`
	UserType string `json:"user_type" validate:"required,oneof=admin user"`
	Password string `json:"password" validate:"required,min=8"`
}

// UpdateUserRequest represents the request to update an existing user
type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}
