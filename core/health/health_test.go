package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestReadyRedisDownFails(t *testing.T) {
	h := &Handler{db: fakePinger{}, redis: fakePinger{err: errors.New("down")}}
	c, w := newTestContext("GET", "/v1/health/ready")
	h.Ready(c)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when configured redis is down", w.Code)
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
