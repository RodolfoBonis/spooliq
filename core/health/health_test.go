package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
)

type fakePinger struct {
	err error
}

func (f fakePinger) PingContext(_ context.Context) error { return f.err }

func newTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func TestLiveAlwaysOK(t *testing.T) {
	h := &Handler{}
	c, w := newTestContext("GET", "/v1/health/live")
	h.Live(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestLegacyCheckPlainText(t *testing.T) {
	h := &Handler{}
	c, w := newTestContext("GET", "/v1/health_check")
	h.LegacyCheck(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Body.String() != "This Service is Healthy" {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestReadyAllOK(t *testing.T) {
	h := &Handler{db: fakePinger{}, redis: fakePinger{}}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	checks := decodeChecks(t, w.Body.Bytes())
	if checks["database"] != "ok" || checks["redis"] != "ok" {
		t.Errorf("checks = %v", checks)
	}
}

func TestReadyRedisSkippedWhenNil(t *testing.T) {
	h := &Handler{db: fakePinger{}, redis: nil}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (redis skipped must not fail)", w.Code)
	}
	checks := decodeChecks(t, w.Body.Bytes())
	if checks["redis"] != "skipped" {
		t.Errorf("redis = %q, want skipped", checks["redis"])
	}
	if checks["database"] != "ok" {
		t.Errorf("database = %q, want ok", checks["database"])
	}
}

func TestReadyDatabaseDown(t *testing.T) {
	h := &Handler{db: fakePinger{err: errors.New("down")}, redis: fakePinger{}}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	checks := decodeChecks(t, w.Body.Bytes())
	if checks["database"] != "error" {
		t.Errorf("database = %q, want error", checks["database"])
	}
}

// A Redis outage degrades the service but must not make it unready: the cache
// layer fails open, so taking the pod out of rotation would cause an outage.
func TestReadyRedisDownIsDegradedNotUnready(t *testing.T) {
	h := &Handler{db: fakePinger{}, redis: fakePinger{err: errors.New("down")}}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 when only redis is down", w.Code)
	}
	if checks := decodeChecks(t, w.Body.Bytes()); checks["redis"] != "error" {
		t.Errorf("redis = %q, want error", checks["redis"])
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Status != "degraded" {
		t.Errorf("status = %q (err %v), want degraded", body.Status, err)
	}
}

// The Redis client is resolved at probe time (it is initialized after the
// handler is built); a resolver returning nil reports "skipped".
func TestReadyResolvesRedisLazily(t *testing.T) {
	h := NewHandler(nil, func() *redis.Client { return nil }, nil)
	h.db = fakePinger{}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if checks := decodeChecks(t, w.Body.Bytes()); checks["redis"] != "skipped" {
		t.Errorf("redis = %q, want skipped when the client is not initialized", checks["redis"])
	}
}

func decodeChecks(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var body struct {
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("invalid body: %v", err)
	}
	return body.Checks
}
