package errors

import "net/http"

// APIError is the canonical error type for HTTP handlers. It carries the HTTP
// status, a stable snake_case code, a user-facing pt-BR message and optional
// per-field validation messages.
//
// Handlers should return an APIError (via the constructors below) and let
// errors.Respond / errors.AbortWith serialize it. The wire format is:
//
//	{
//	  "error":   "<pt-BR message>",
//	  "message": "<pt-BR message>",
//	  "code":    "<snake_case>",
//	  "fields":  {"field": "msg"}   // omitted when empty
//	}
//
// "error" and "message" intentionally carry the same text: "error" preserves
// backward compatibility with older clients while "message" is the forward
// name used across the API.
type APIError struct {
	// Status is the HTTP status code. Not serialized in the body.
	Status int `json:"-"`
	// Code is a stable, snake_case machine-readable identifier.
	Code string `json:"code"`
	// Message is the user-facing pt-BR message.
	Message string `json:"message"`
	// Fields holds per-field validation messages. Omitted when empty.
	Fields map[string]string `json:"fields,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// errorBody is the serialized envelope. It duplicates the message into both
// "error" and "message" to satisfy the documented contract.
type errorBody struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Code    string            `json:"code"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func (e *APIError) body() errorBody {
	return errorBody{
		Error:   e.Message,
		Message: e.Message,
		Code:    e.Code,
		Fields:  e.Fields,
	}
}

// newAPIError is the shared constructor used by the helpers below.
func newAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

// BadRequest builds a 400 Bad Request APIError.
func BadRequest(code, message string) *APIError {
	return newAPIError(http.StatusBadRequest, code, message)
}

// Unauthorized builds a 401 Unauthorized APIError.
func Unauthorized(code, message string) *APIError {
	return newAPIError(http.StatusUnauthorized, code, message)
}

// PaymentRequired builds a 402 Payment Required APIError.
func PaymentRequired(code, message string) *APIError {
	return newAPIError(http.StatusPaymentRequired, code, message)
}

// Forbidden builds a 403 Forbidden APIError.
func Forbidden(code, message string) *APIError {
	return newAPIError(http.StatusForbidden, code, message)
}

// NotFoundErr builds a 404 Not Found APIError. It is named NotFoundErr (not
// NotFound) to avoid colliding with the existing AppError helper NotFound.
func NotFoundErr(code, message string) *APIError {
	return newAPIError(http.StatusNotFound, code, message)
}

// Conflict builds a 409 Conflict APIError.
func Conflict(code, message string) *APIError {
	return newAPIError(http.StatusConflict, code, message)
}

// Validation builds a 400 Bad Request APIError carrying per-field messages.
// The code is fixed to "validation_error".
func Validation(fields map[string]string) *APIError {
	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    CodeValidationError,
		Message: "Erro de validação",
		Fields:  fields,
	}
}

// Internal builds a generic 500 Internal Server Error APIError. The message is
// intentionally generic and never leaks the underlying cause.
func Internal() *APIError {
	return newAPIError(http.StatusInternalServerError, CodeInternalError, genericInternalMessage)
}

// Gone builds a 410 Gone APIError.
func Gone(code, message string) *APIError {
	return newAPIError(http.StatusGone, code, message)
}

// TooManyRequests builds a 429 Too Many Requests APIError.
func TooManyRequests(code, message string) *APIError {
	return newAPIError(http.StatusTooManyRequests, code, message)
}
