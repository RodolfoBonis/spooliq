package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func decodeBody(t *testing.T, body string) errorBody {
	t.Helper()
	var eb errorBody
	if err := json.Unmarshal([]byte(body), &eb); err != nil {
		t.Fatalf("invalid JSON body %q: %v", body, err)
	}
	return eb
}

func TestRespondMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string // substring expected in message
	}{
		{
			name:       "APIError used as-is",
			err:        Conflict("email_taken", "E-mail já cadastrado"),
			wantStatus: http.StatusConflict,
			wantCode:   "email_taken",
			wantMsg:    "E-mail já cadastrado",
		},
		{
			name:       "AppError maps status and code",
			err:        NewAppError(entities.ErrNotFound, "Cliente não encontrado", nil, nil),
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
			wantMsg:    "Cliente não encontrado",
		},
		{
			name:       "AppError forbidden",
			err:        NewAppError(entities.ErrForbidden, "Acesso negado", nil, nil),
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
			wantMsg:    "Acesso negado",
		},
		{
			name:       "gorm record not found",
			err:        gorm.ErrRecordNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
			wantMsg:    "não encontrado",
		},
		{
			name:       "wrapped gorm record not found",
			err:        fmt.Errorf("load budget: %w", gorm.ErrRecordNotFound),
			wantStatus: http.StatusNotFound,
			wantCode:   CodeNotFound,
			wantMsg:    "não encontrado",
		},
		{
			name:       "unmapped error is generic 500",
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusInternalServerError,
			wantCode:   CodeInternalError,
			wantMsg:    genericInternalMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			Respond(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			eb := decodeBody(t, w.Body.String())
			if eb.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", eb.Code, tt.wantCode)
			}
			if !strings.Contains(eb.Message, tt.wantMsg) {
				t.Errorf("message = %q, want substring %q", eb.Message, tt.wantMsg)
			}
			// error and message carry the same text.
			if eb.Error != eb.Message {
				t.Errorf("error %q != message %q", eb.Error, eb.Message)
			}
		})
	}
}

func TestRespondInternalDoesNotLeak(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	secret := "connection string with password=supersecret"
	Respond(c, stderr(secret))

	body := w.Body.String()
	if strings.Contains(body, "supersecret") {
		t.Fatalf("internal error leaked into body: %s", body)
	}
	eb := decodeBody(t, body)
	if eb.Message != genericInternalMessage {
		t.Errorf("message = %q, want generic", eb.Message)
	}
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func stderr(s string) error { return simpleErr(s) }

type validatedPayload struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=18"`
	Bio   string `json:"bio" validate:"max=10"`
}

func TestRespondValidationFieldsPtBR(t *testing.T) {
	payload := validatedPayload{
		Name:  "",
		Email: "not-an-email",
		Age:   10,
		Bio:   "way too long bio field",
	}
	err := validation.Validate(payload)
	if err == nil {
		t.Fatal("expected validation error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	Respond(c, err)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	eb := decodeBody(t, w.Body.String())
	if eb.Code != CodeValidationError {
		t.Errorf("code = %q, want %q", eb.Code, CodeValidationError)
	}
	if eb.Fields["name"] != "obrigatório" {
		t.Errorf("name field = %q, want obrigatório", eb.Fields["name"])
	}
	if eb.Fields["email"] != "deve ser um e-mail válido" {
		t.Errorf("email field = %q", eb.Fields["email"])
	}
	if eb.Fields["age"] != "deve ser maior ou igual a 18" {
		t.Errorf("age field = %q", eb.Fields["age"])
	}
	if !strings.Contains(eb.Fields["bio"], "no máximo 10 caracteres") {
		t.Errorf("bio field = %q", eb.Fields["bio"])
	}
}

func TestValidationUsesJSONFieldNames(t *testing.T) {
	err := validation.Validate(struct {
		FullName string `json:"full_name" validate:"required"`
	}{})
	if err == nil {
		t.Fatal("expected error")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", nil)
	Respond(c, err)
	eb := decodeBody(t, w.Body.String())
	if _, ok := eb.Fields["full_name"]; !ok {
		t.Errorf("expected json field name full_name, got %+v", eb.Fields)
	}
}

func TestAPIErrorConstructors(t *testing.T) {
	tests := []struct {
		err        *APIError
		wantStatus int
	}{
		{BadRequest("c", "m"), http.StatusBadRequest},
		{Unauthorized("c", "m"), http.StatusUnauthorized},
		{PaymentRequired("c", "m"), http.StatusPaymentRequired},
		{Forbidden("c", "m"), http.StatusForbidden},
		{NotFoundErr("c", "m"), http.StatusNotFound},
		{Conflict("c", "m"), http.StatusConflict},
		{Internal(), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if tt.err.Status != tt.wantStatus {
			t.Errorf("status = %d, want %d", tt.err.Status, tt.wantStatus)
		}
		if tt.err.Error() != tt.err.Message {
			t.Errorf("Error() should return Message")
		}
	}
}

func TestFieldsOmittedWhenEmpty(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	Respond(c, BadRequest("x", "y"))
	if strings.Contains(w.Body.String(), "fields") {
		t.Errorf("fields must be omitted when empty: %s", w.Body.String())
	}
}
