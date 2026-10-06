package users

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopLogger satisfies logger.Logger without side effects.
type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

// stubUserRepo is a minimal in-memory UserRepository for list tests. It applies
// search (name/email) and pagination the same way the real repository does.
type stubUserRepo struct {
	users []*entities.UserEntity
}

func (r *stubUserRepo) FindAll(_ context.Context, organizationID string, q helpers.ListQuery) ([]*entities.UserEntity, int64, error) {
	var matched []*entities.UserEntity
	for _, u := range r.users {
		if u.OrganizationID != organizationID {
			continue
		}
		if q.Search != "" {
			needle := strings.ToLower(q.Search)
			if !strings.Contains(strings.ToLower(u.Name), needle) && !strings.Contains(strings.ToLower(u.Email), needle) {
				continue
			}
		}
		matched = append(matched, u)
	}
	total := int64(len(matched))
	off := q.Offset()
	if off > len(matched) {
		off = len(matched)
	}
	end := len(matched)
	if q.Limit() > 0 {
		end = off + q.Limit()
		if end > len(matched) {
			end = len(matched)
		}
	}
	return matched[off:end], total, nil
}

func (r *stubUserRepo) FindByID(context.Context, uuid.UUID, string) (*entities.UserEntity, error) {
	return nil, nil
}
func (r *stubUserRepo) FindByEmail(context.Context, string) (*entities.UserEntity, error) {
	return nil, nil
}
func (r *stubUserRepo) FindByKeycloakUserID(context.Context, string) (*entities.UserEntity, error) {
	return nil, nil
}
func (r *stubUserRepo) FindOwner(context.Context, string) (*entities.UserEntity, error) {
	return nil, nil
}
func (r *stubUserRepo) Create(context.Context, *entities.UserEntity) error             { return nil }
func (r *stubUserRepo) Update(context.Context, uuid.UUID, string, *entities.UserEntity) error {
	return nil
}
func (r *stubUserRepo) Delete(context.Context, uuid.UUID, string) error { return nil }

type listEnvelope struct {
	Data       []json.RawMessage `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

func listHandler(repo *stubUserRepo) *Handler {
	return NewUserHandler(nil, usecases.NewListUsersUseCase(repo, noopLogger{}), nil, nil, nil)
}

func ctxWith(path string, org string, userRoles []string) (*httptest.ResponseRecorder, *gin.Context) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	if org != "" {
		c.Set("organization_id", org)
	}
	c.Set("user_roles", userRoles)
	return w, c
}

const orgA = "org-a"

func TestListUsers_Envelope(t *testing.T) {
	repo := &stubUserRepo{users: []*entities.UserEntity{
		{ID: uuid.New(), OrganizationID: orgA, Name: "Ana", Email: "ana@x.com"},
		{ID: uuid.New(), OrganizationID: orgA, Name: "Bruno", Email: "bruno@x.com"},
		{ID: uuid.New(), OrganizationID: "other", Name: "Carla", Email: "carla@x.com"},
	}}
	h := listHandler(repo)

	w, c := ctxWith("/users", orgA, []string{roles.OwnerRole})
	h.ListUsers(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(2), env.Total)
	assert.Len(t, env.Data, 2)
	assert.Equal(t, 20, env.PageSize)
}

func TestListUsers_SearchByEmail(t *testing.T) {
	repo := &stubUserRepo{users: []*entities.UserEntity{
		{ID: uuid.New(), OrganizationID: orgA, Name: "Ana", Email: "ana@x.com"},
		{ID: uuid.New(), OrganizationID: orgA, Name: "Bruno", Email: "bruno@y.com"},
	}}
	h := listHandler(repo)

	w, c := ctxWith("/users?q=bruno", orgA, []string{roles.OrgAdminRole})
	h.ListUsers(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env listEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(1), env.Total)
	assert.Len(t, env.Data, 1)
}

func TestListUsers_ForbiddenCode(t *testing.T) {
	repo := &stubUserRepo{}
	h := listHandler(repo)

	w, c := ctxWith("/users", orgA, []string{roles.UserRole})
	h.ListUsers(c)

	require.Equal(t, http.StatusForbidden, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "users_list_forbidden", body.Code)
}
