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
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrEntity, message, context, nil)
}

// EnvironmentError creates a new environment error.
func EnvironmentError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrEnvironment, message, context, nil)
}

// MiddlewareError creates a new middleware error.
func MiddlewareError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrMiddleware, message, context, nil)
}

// ModelError creates a new model error.
func ModelError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrModel, message, context, nil)
}

// RepositoryError creates a new repository error.
func RepositoryError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrRepository, message, context, nil)
}

// RootError creates a new root error.
func RootError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrRoot, message, context, nil)
}

// ServiceError creates a new service error.
func ServiceError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrService, message, context, nil)
}

// UsecaseError creates a new use case error.
func UsecaseError(message string, ctx ...map[string]interface{}) *AppError {
	var context map[string]interface{}
	if len(ctx) > 0 {
		context = ctx[0]
	}
	return NewAppError(entities.ErrUsecase, message, context, nil)
}
