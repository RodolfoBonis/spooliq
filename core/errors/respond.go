package errors

import (
	"context"
	stderrors "errors"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// Stable, snake_case error codes used by the envelope. Feature code should
// prefer specific codes (e.g. "invalid_customer_id") but these cover the
// framework-level cases produced by Respond itself.
const (
	CodeNotFound         = "not_found"
	CodeValidationError  = "validation_error"
	CodeInvalidRequest   = "invalid_request"
	CodeInternalError    = "internal_error"
	CodeRouteNotFound    = "route_not_found"
	CodeMethodNotAllowed = "method_not_allowed"
)

// genericInternalMessage is the pt-BR message returned for any unmapped error.
// It never includes the underlying cause so internal details do not leak.
const genericInternalMessage = "Erro interno do servidor"

// appErrorTypeToCode maps the legacy AppError types to snake_case codes so that
// existing *AppError values returned by use cases serialize into the new
// envelope with a meaningful code.
var appErrorTypeToCode = map[entities.AppErrorType]string{
	entities.ErrDatabase:           "database_error",
	entities.ErrRepository:         "repository_error",
	entities.ErrUsecase:            "usecase_error",
	entities.ErrEntity:             "bad_request",
	entities.ErrModel:              "bad_request",
	entities.ErrService:            CodeInternalError,
	entities.ErrMiddleware:         "middleware_error",
	entities.ErrRoot:               CodeInternalError,
	entities.ErrEnvironment:        CodeInternalError,
	entities.ErrNotFound:           CodeNotFound,
	entities.ErrInvalidToken:       "invalid_token",
	entities.ErrInvalidCredentials: "invalid_credentials",
	entities.ErrUnauthorized:       "unauthorized",
	entities.ErrConflict:           "conflict",
	entities.ErrForbidden:          "forbidden",
	entities.ErrPaymentRequired:    "payment_required",
	entities.ErrExternalService:    "external_service_error",
}

// pkgLogger is a package-level logger used to record internal (500) errors.
// It is wired at startup via SetLogger. When nil, logging is skipped so that
// Respond remains usable in tests without a logger.
var pkgLogger logger.Logger

// SetLogger wires the logger used to record internal errors. It is called once
// at application startup (see app wiring). Safe to leave unset in tests.
func SetLogger(l logger.Logger) {
	pkgLogger = l
}

// Respond maps err to the standard envelope and writes it to the response.
// It inspects, in order:
//  1. *APIError          -> used as-is.
//  2. *AppError          -> its HTTPStatus with a code derived from its type.
//  3. gorm.ErrRecordNotFound -> 404 not_found.
//  4. validator.ValidationErrors -> 400 validation_error with pt-BR fields.
//  5. JSON/binding errors -> 400 invalid_request.
//  6. anything else      -> 500 internal_error (the real error is logged, never
//     written to the body).
func Respond(c *gin.Context, err error) {
	apiErr := Map(c.Request.Context(), err)
	c.JSON(apiErr.Status, apiErr.body())
}

// AbortWith behaves like Respond but also aborts the gin handler chain, so no
// further handlers or middleware run after the error is written.
func AbortWith(c *gin.Context, err error) {
	apiErr := Map(c.Request.Context(), err)
	c.AbortWithStatusJSON(apiErr.Status, apiErr.body())
}

// Map converts any error into an *APIError following the documented precedence.
// It logs internal (500) errors from the unmapped bucket using the package
// logger. ctx is used only for log correlation and may be context.Background()
// in non-request paths.
func Map(ctx context.Context, err error) *APIError {
	if err == nil {
		return Internal()
	}

	// 1. Already an APIError.
	var apiErr *APIError
	if stderrors.As(err, &apiErr) {
		return apiErr
	}

	// 2. Legacy *AppError. Status and code are mapped; the pt-BR message is
	//    preserved as-is (callers already set user-facing messages).
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return fromAppError(appErr)
	}

	// 3. Record not found.
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return NotFoundErr(CodeNotFound, "Recurso não encontrado")
	}

	// 4. Validation errors.
	var verrs validator.ValidationErrors
	if stderrors.As(err, &verrs) {
		return Validation(translateValidationErrors(verrs))
	}

	// 5. JSON / binding errors.
	if isBindingError(err) {
		return BadRequest(CodeInvalidRequest, "Requisição inválida")
	}

	// 6. Unmapped -> internal. Log the real error, return a generic body.
	logInternal(ctx, err)
	return Internal()
}

func fromAppError(appErr *AppError) *APIError {
	code, ok := appErrorTypeToCode[appErr.Type]
	if !ok {
		code = CodeInternalError
	}
	return &APIError{
		Status:  appErr.HTTPStatus(),
		Code:    code,
		Message: appErr.Message,
		Fields:  stringifyFields(appErr.Fields),
	}
}

// stringifyFields converts an AppError's loosely-typed Fields into the
// string->string shape used by the envelope. Non-string values are dropped so
// internal diagnostic context never leaks into client responses.
func stringifyFields(fields map[string]interface{}) map[string]string {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]string, len(fields))
	for k, v := range fields {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func logInternal(ctx context.Context, err error) {
	if pkgLogger == nil || err == nil {
		return
	}
	pkgLogger.LogError(ctx, "Unhandled internal error", err)
}
