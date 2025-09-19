package entities

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// RequestRegisterEntity represents the registration request payload.
// @Description Registration request data
// @Example {"email": "user@example.com", "password": "SecurePass123", "firstName": "John", "lastName": "Doe"}
type RequestRegisterEntity struct {
	// User email address
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
	// User password (minimum 8 characters)
	Password string `json:"password" validate:"required,min=8" example:"SecurePass123"`
	// User first name
	FirstName string `json:"firstName" validate:"required,min=2" example:"John"`
	// User last name
	LastName string `json:"lastName" validate:"required,min=2" example:"Doe"`
}

// Validate validates the registration entity
func (r *RequestRegisterEntity) Validate() error {
	validate := validator.New()

	// Validate struct
	if err := validate.Struct(r); err != nil {
		return err
	}

	// Additional business validations
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)

	return nil
}

// RegisterResponseEntity represents the registration response
// @Description Registration response data
type RegisterResponseEntity struct {
	// Success message
	Message string `json:"message" example:"User registered successfully"`
	// User email
	Email string `json:"email" example:"user@example.com"`
	// User ID (UUID)
	UserID string `json:"userID" example:"uuid"`
}

// ForgotPasswordRequestEntity represents the forgot password request payload.
// @Description Forgot password request data
// @Example {"email": "user@example.com"}
type ForgotPasswordRequestEntity struct {
	// User email address
	Email string `json:"email" validate:"required,email" example:"user@example.com"`
}

// Validate validates the forgot password entity
func (r *ForgotPasswordRequestEntity) Validate() error {
	validate := validator.New()

	// Validate struct
	if err := validate.Struct(r); err != nil {
		return err
	}

	// Additional business validations
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))

	return nil
}

// ForgotPasswordResponseEntity represents the forgot password response
// @Description Forgot password response data
type ForgotPasswordResponseEntity struct {
	// Success message
	Message string `json:"message" example:"Password reset instructions have been sent to your email"`
	// User email
	Email string `json:"email" example:"user@example.com"`
}