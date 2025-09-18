package errors

// HTTPError represents an HTTP error.
type HTTPError struct {
	StatusCode int                    `json:"code"`
	Message    string                 `json:"message"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Cause      string                 `json:"cause,omitempty"`
}

// NewHTTPError creates a new HttpError with status, message and optional stack/context.
func NewHTTPError(statusCode int, message string, stack ...string) *HTTPError {
	h := &HTTPError{
		StatusCode: statusCode,
		Message:    message,
	}
	if len(stack) > 0 {
		h.StackTrace = stack[0]
	}
	return h
}

// FromAppError creates a HttpError from an AppError.
func FromAppError(err *AppError) *HTTPError {
	return &HTTPError{
		StatusCode: err.HTTPStatus(),
		Message:    err.Message,
		Context:    err.Fields,
		Cause:      unwrapCause(err.Cause),
	}
}

// ToMap returns a map for structured logging.
func (e *HTTPError) ToMap() map[string]interface{} {
	fields := map[string]interface{}{
		"code":    e.StatusCode,
		"message": e.Message,
	}
	if e.StackTrace != "" {
		fields["stack_trace"] = e.StackTrace
	}
	if e.Context != nil {
		fields["context"] = e.Context
	}
	if e.Cause != "" {
		fields["cause"] = e.Cause
	}
	return fields
}

// Helper to unwrap cause from error chain.
func unwrapCause(err error) string {
	if err == nil {
		return ""
	}
	if u, ok := err.(interface{ Unwrap() error }); ok {
		if cause := u.Unwrap(); cause != nil {
			return cause.Error()
		}
	}
	return ""
}
