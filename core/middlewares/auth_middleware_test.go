package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/core/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() { gin.SetMode(gin.TestMode) }

// newTestContext returns a gin context backed by a recorder, with a request so
// c.Request.Context() is available to the code under test.
func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c, rec
}

func TestAuthorizeRole_Allows(t *testing.T) {
	c, rec := newTestContext()
	userRoles := types.Array{roles.UserRole}

	matched, ok := authorizeRole(c, &logger.NoopLogger{}, userRoles, []string{roles.OwnerRole, roles.UserRole})

	assert.True(t, ok)
	assert.Equal(t, roles.UserRole, matched)
	assert.False(t, c.IsAborted())
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthorizeRole_ForbiddenWhenMissingRole(t *testing.T) {
	c, rec := newTestContext()
	// Valid token (user has a role) but not one of the required roles.
	userRoles := types.Array{roles.UserRole}

	matched, ok := authorizeRole(c, &logger.NoopLogger{}, userRoles, []string{roles.OwnerRole, roles.OrgAdminRole})

	assert.False(t, ok)
	assert.Empty(t, matched)
	assert.True(t, c.IsAborted())

	// Must be 403 (authorization), never 401 (authentication).
	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Code    string `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "insufficient_role", body.Code)
	assert.Equal(t, "Você não tem permissão para realizar esta ação", body.Message)
	assert.Equal(t, body.Message, body.Error)
}
