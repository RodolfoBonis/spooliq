package activity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
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

// stubRepo records the filter it received and returns a fixed page.
type stubRepo struct {
	lastFilter *entities.ActivityFilter
}

func (r *stubRepo) Create(*entities.ActivityEntity) error { return nil }
func (r *stubRepo) FindByOrganization(filter *entities.ActivityFilter) (*entities.PaginatedActivities, error) {
	r.lastFilter = filter
	return &entities.PaginatedActivities{
		Activities: []entities.ActivityEntity{{EntityID: "a"}, {EntityID: "b"}},
		Total:      5,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: 3,
	}, nil
}
func (r *stubRepo) FindRecentByOrganization(string, int) ([]entities.ActivityEntity, error) {
	return nil, nil
}

// ListActivities returns the standard envelope keyed by "data" (not "activities").
func TestListActivities_Envelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubRepo{}
	svc := usecases.NewActivityService(repo, noopLogger{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/activities?page=2&page_size=2&entity_type=preset", nil)
	c.Set("organization_id", "org-a")

	svc.ListActivities(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// Raw body must carry "data" and must NOT carry the legacy "activities" key.
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	_, hasData := raw["data"]
	_, hasLegacy := raw["activities"]
	assert.True(t, hasData, "envelope must use data key")
	assert.False(t, hasLegacy, "legacy activities key must be gone")

	var env struct {
		Data       []json.RawMessage `json:"data"`
		Total      int64             `json:"total"`
		Page       int               `json:"page"`
		PageSize   int               `json:"page_size"`
		TotalPages int               `json:"total_pages"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(5), env.Total)
	assert.Len(t, env.Data, 2)
	assert.Equal(t, 2, env.Page)
	assert.Equal(t, 2, env.PageSize)

	// Filters are threaded through to the repository.
	require.NotNil(t, repo.lastFilter)
	require.NotNil(t, repo.lastFilter.EntityType)
	assert.Equal(t, entities.ActivityEntityType("preset"), *repo.lastFilter.EntityType)
}

// Missing organization yields a 401 with a stable code.
func TestListActivities_MissingOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := usecases.NewActivityService(&stubRepo{}, noopLogger{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/activities", nil)

	svc.ListActivities(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "organization_id_missing", body.Code)
}
