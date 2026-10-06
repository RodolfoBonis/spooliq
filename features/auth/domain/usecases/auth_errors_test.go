package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

func newCtx(method, path, body string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return w, c
}

// A malformed login body returns 400 with the stable invalid_request code and
// the standard envelope (code + message), never a leaked parser error.
func TestValidateLogin_InvalidBodyCode(t *testing.T) {
	uc := &authUseCaseImpl{Logger: noopLogger{}}

	w, c := newCtx(http.MethodPost, "/login", "{not-json")
	uc.ValidateLogin(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body struct {
		Code    string `json:"code"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "invalid_request", body.Code)
	assert.NotEmpty(t, body.Message)
	assert.Equal(t, body.Error, body.Message)
}

// Logout without an Authorization header returns 401 missing_token.
func TestLogout_MissingTokenCode(t *testing.T) {
	uc := &authUseCaseImpl{Logger: noopLogger{}}

	w, c := newCtx(http.MethodPost, "/logout", "")
	uc.Logout(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "missing_token", body.Code)
}
