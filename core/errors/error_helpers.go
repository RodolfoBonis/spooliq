package errors

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/entities"
)

// BadRequestError creates a 400 Bad Request error
func BadRequestError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrEntity,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// UnauthorizedError creates a 401 Unauthorized error
func UnauthorizedError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrUnauthorized,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// ForbiddenError creates a 403 Forbidden error (also maps to 403 via custom handling)
func ForbiddenError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrUnauthorized,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// NotFound creates a 404 Not Found error
func NotFound(message string) *AppError {
	return &AppError{
		Type:    entities.ErrNotFound,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// ConflictError creates a 409 Conflict error
func ConflictError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrConflict,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// InternalServerError creates a 500 Internal Server Error
func InternalServerError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrService,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// ExternalServiceError creates a 502 Bad Gateway error (for external service failures)
func ExternalServiceError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrService,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// PaymentRequiredError creates a 402 Payment Required error
func PaymentRequiredError(message string) *AppError {
	return &AppError{
		Type:    entities.ErrEntity,
		Message: message,
		Fields:  nil,
		Cause:   nil,
	}
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == entities.ErrNotFound || appErr.HTTPStatus() == http.StatusNotFound
	}
	return false
}

