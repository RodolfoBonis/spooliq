package entities

import (
	"errors"
	"regexp"
	"time"

	"github.com/RodolfoBonis/spooliq/core/roles"
)

// User represents a user in the domain
type User struct {
	ID         string            `json:"id"`
	Username   string            `json:"username"`
	Email      string            `json:"email"`
	FirstName  string            `json:"first_name"`
	LastName   string            `json:"last_name"`
	Enabled    bool              `json:"enabled"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Roles      []string          `json:"roles"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// UserCreateRequest represents data needed to create a new user
type UserCreateRequest struct {
	Username          string `json:"username" validate:"required,min=3,max=50"`
	Email             string `json:"email" validate:"required,email"`
	FirstName         string `json:"first_name" validate:"required,min=1,max=100"`
	LastName          string `json:"last_name" validate:"required,min=1,max=100"`
	Password          string `json:"password" validate:"required,min=8"`
	Enabled           bool   `json:"enabled"`
	TemporaryPassword bool   `json:"temporary_password"`
}

// UserUpdateRequest represents data that can be updated for a user
type UserUpdateRequest struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

// UserListQuery represents query parameters for listing users
type UserListQuery struct {
	Search string `json:"search,omitempty"`
	First  int    `json:"first,omitempty"`
	Max    int    `json:"max,omitempty"`
}

// UserStats represents user statistics in the domain
type UserStats struct {
	Total     int `json:"total"`
	Active    int `json:"active"`
	Inactive  int `json:"inactive"`
	Suspended int `json:"suspended"`
	Admins    int `json:"admins"`
}

// Business validation methods

// ValidateCreate validates user creation request
func (req *UserCreateRequest) ValidateCreate() error {
	if err := req.validateUsername(); err != nil {
		return err
	}
	if err := req.validateEmail(); err != nil {
		return err
	}
	if err := req.validatePassword(); err != nil {
		return err
	}
	return nil
}

// ValidateUpdate validates user update request
func (req *UserUpdateRequest) ValidateUpdate() error {
	if req.Email != nil {
		if err := validateEmailFormat(*req.Email); err != nil {
			return err
		}
	}
	return nil
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	return u.FirstName + " " + u.LastName
}

// HasRole checks if user has a specific role
func (u *User) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if user has admin role
func (u *User) IsAdmin() bool {
	return u.HasRole(roles.AdminRole) || u.HasRole("Admin")
}

// CanModifyUser checks if current user can modify target user
func (u *User) CanModifyUser(targetUserID string) bool {
	// Admin can modify anyone except themselves
	if u.IsAdmin() && u.ID != targetUserID {
		return true
	}
	// Users can only modify themselves
	return u.ID == targetUserID
}

// Private validation helpers

func (req *UserCreateRequest) validateUsername() error {
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return errors.New("username must be between 3 and 50 characters")
	}

	// Username must contain only alphanumeric characters, dots, underscores, and hyphens
	matched, _ := regexp.MatchString("^[a-zA-Z0-9._-]+$", req.Username)
	if !matched {
		return errors.New("username can only contain letters, numbers, dots, underscores, and hyphens")
	}

	return nil
}

func (req *UserCreateRequest) validateEmail() error {
	return validateEmailFormat(req.Email)
}

func validateEmailFormat(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func (req *UserCreateRequest) validatePassword() error {
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	// Check for at least one uppercase letter
	hasUpper, _ := regexp.MatchString(`[A-Z]`, req.Password)
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	// Check for at least one lowercase letter
	hasLower, _ := regexp.MatchString(`[a-z]`, req.Password)
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	// Check for at least one number
	hasNumber, _ := regexp.MatchString(`[0-9]`, req.Password)
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}

	return nil
}

// Common domain errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidUserData   = errors.New("invalid user data")
	ErrUnauthorized      = errors.New("unauthorized to perform this action")
	ErrPasswordTooWeak   = errors.New("password does not meet security requirements")
)
