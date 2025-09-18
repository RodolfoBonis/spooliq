package errors

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/entities"
)

// Error is the base interface for all custom errors in the system.
type Error interface {
	error
	Code() int
	Message() string
	StackTrace() string
	Context() map[string]interface{}
	Unwrap() error
	ToLogFields() map[string]interface{}
	ToHTTPError() *HTTPError
}

// AppError representa um erro de aplicação padronizado.
type AppError struct {
	Type    entities.AppErrorType
	Message string
	Fields  map[string]interface{}
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}

// HTTPStatus returns the HTTP status code for the AppError.
func (e *AppError) HTTPStatus() int {
	if status, ok := entities.AppErrorTypeToHTTP[e.Type]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// NewAppError cria um novo erro padronizado.
func NewAppError(errType entities.AppErrorType, msg string, fields map[string]interface{}, cause error) *AppError {
	if msg == "" {
		msg = entities.AppErrorTypeToString[errType]
	}
	return &AppError{
		Type:    errType,
		Message: msg,
		Fields:  fields,
		Cause:   cause,
	}
}

// ToLogFields returns a map with all error details for structured logging.
func (e *AppError) ToLogFields() map[string]interface{} {
	fields := map[string]interface{}{
		"error_code":    e.Type,
		"error_message": e.Message,
	}
	for k, v := range e.Fields {
		fields[k] = v
	}
	if e.Cause != nil {
		fields["cause"] = e.Cause.Error()
	}
	return fields
}

// ToHTTPError converts an AppError to an HTTP error.
func (e *AppError) ToHTTPError() *HTTPError {
	return NewHTTPError(e.HTTPStatus(), e.Message)
}

// EntityError creates a new entity error.
func EntityError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrEntity, message, ctx[0], nil)
}

// EnvironmentError creates a new environment error.
func EnvironmentError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrEnvironment, message, ctx[0], nil)
}

// MiddlewareError creates a new middleware error.
func MiddlewareError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrMiddleware, message, ctx[0], nil)
}

// ModelError creates a new model error.
func ModelError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrModel, message, ctx[0], nil)
}

// RepositoryError creates a new repository error.
func RepositoryError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrRepository, message, ctx[0], nil)
}

// RootError creates a new root error.
func RootError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrRoot, message, ctx[0], nil)
}

// ServiceError creates a new service error.
func ServiceError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrService, message, ctx[0], nil)
}

// UsecaseError creates a new use case error.
func UsecaseError(message string, ctx ...map[string]interface{}) *AppError {
	return NewAppError(entities.ErrUsecase, message, ctx[0], nil)
}
