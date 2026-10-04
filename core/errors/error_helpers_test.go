package errors

import (
	"net/http"
	"testing"
)

// TestErrorHelperStatusCodes guards the HTTP status code each helper maps to.
// These codes are part of the public API contract, and three of them were
// previously wrong (403->401, 402->400, 502->500).
func TestErrorHelperStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        *AppError
		wantStatus int
	}{
		{"BadRequestError", BadRequestError("bad"), http.StatusBadRequest},
		{"UnauthorizedError", UnauthorizedError("unauth"), http.StatusUnauthorized},
		{"ForbiddenError", ForbiddenError("forbidden"), http.StatusForbidden},
		{"NotFound", NotFound("missing"), http.StatusNotFound},
		{"ConflictError", ConflictError("conflict"), http.StatusConflict},
		{"PaymentRequiredError", PaymentRequiredError("pay"), http.StatusPaymentRequired},
		{"InternalServerError", InternalServerError("boom"), http.StatusInternalServerError},
		{"ExternalServiceError", ExternalServiceError("upstream"), http.StatusBadGateway},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.HTTPStatus(); got != tt.wantStatus {
				t.Errorf("%s: HTTPStatus() = %d, want %d", tt.name, got, tt.wantStatus)
			}
			if got := tt.err.ToHTTPError().StatusCode; got != tt.wantStatus {
				t.Errorf("%s: ToHTTPError().StatusCode = %d, want %d", tt.name, got, tt.wantStatus)
			}
			if tt.err.Message == "" {
				t.Errorf("%s: expected message to be preserved", tt.name)
			}
		})
	}
}
