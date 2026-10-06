package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
)

// newTestCache spins up an in-memory Redis (miniredis) wired through the real
// RedisService, so the tests exercise the actual cache read/write/scan paths.
func newTestCache(t *testing.T) (*CacheMiddleware, *miniredis.Miniredis) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	cfg := &config.AppConfig{
		RedisHost: mr.Host(),
		RedisPort: mr.Port(),
		RedisDB:   0,
	}
	rs := services.NewRedisService(&logger.NoopLogger{}, cfg)
	if appErr := rs.Init(); appErr != nil {
		t.Fatalf("failed to init redis service: %v", appErr)
	}

	return NewCacheMiddleware(rs, &logger.NoopLogger{}), mr
}

// orgMiddleware mimics what protectFactory does: it puts organization_id (and a
// user id) into the context before the cache handler runs.
func orgMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("organization_id", c.GetHeader("X-Org"))
		c.Set("user_id", c.GetHeader("X-User"))
		c.Next()
	}
}

func TestGenerateCacheKeyIncludesOrgAndQuery(t *testing.T) {
	cm, _ := newTestCache(t)

	keyFor := func(org string) string {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/filaments?color=red&limit=10", nil)
		c.Set("organization_id", org)
		return cm.generateCacheKey(c, CacheConfig{KeyPrefix: "filaments"})
	}

	keyA := keyFor("org-a")
	keyB := keyFor("org-b")

	if !strings.Contains(keyA, "org:org-a") {
		t.Errorf("key must include organization id, got %q", keyA)
	}
	if !strings.Contains(keyA, "query:color=red&limit=10") {
		t.Errorf("key must include the full query string, got %q", keyA)
	}
	if keyA == keyB {
		t.Errorf("different orgs must produce different keys, both were %q", keyA)
	}
	if !strings.HasPrefix(keyA, "cache:filaments:org:org-a") {
		t.Errorf("key must start with the stable invalidation prefix, got %q", keyA)
	}
}

func TestCacheServesStoredResponseAndIsolatesTenants(t *testing.T) {
	cm, mr := newTestCache(t)

	calls := map[string]int{}
	handler := func(c *gin.Context) {
		org := c.GetHeader("X-Org")
		calls[org]++
		c.JSON(http.StatusOK, gin.H{"org": org})
	}

	router := gin.New()
	router.GET("/filaments", orgMiddleware(), cm.Cache5Min("filaments", handler))

	do := func(org string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/filaments", nil)
		req.Header.Set("X-Org", org)
		router.ServeHTTP(rec, req)
		return rec
	}

	// First request for org-a: miss, handler runs, response cached.
	rec1 := do("org-a")
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("expected MISS on first request, got %q", rec1.Header().Get("X-Cache"))
	}

	// Second identical request: hit, handler NOT called again, same body.
	rec2 := do("org-a")
	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected HIT on second request, got %q", rec2.Header().Get("X-Cache"))
	}
	if calls["org-a"] != 1 {
		t.Errorf("handler should be called once for org-a, got %d", calls["org-a"])
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Errorf("cached body differs: %q vs %q", rec1.Body.String(), rec2.Body.String())
	}

	// Different tenant must not read org-a's cache entry.
	rec3 := do("org-b")
	if rec3.Header().Get("X-Cache") != "MISS" {
		t.Errorf("org-b must not hit org-a cache, got %q", rec3.Header().Get("X-Cache"))
	}
	if calls["org-b"] != 1 {
		t.Errorf("handler should be called once for org-b, got %d", calls["org-b"])
	}
	if !strings.Contains(rec3.Body.String(), "org-b") {
		t.Errorf("org-b must receive its own response, got %q", rec3.Body.String())
	}

	// Two distinct cache entries must exist.
	if n := len(mr.Keys()); n != 2 {
		t.Errorf("expected 2 cache entries (one per org), got %d: %v", n, mr.Keys())
	}
}

func TestCacheOnlyStores200(t *testing.T) {
	cm, mr := newTestCache(t)

	router := gin.New()
	router.GET("/filaments", orgMiddleware(), cm.Cache5Min("filaments", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad"})
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/filaments", nil)
	req.Header.Set("X-Org", "org-a")
	router.ServeHTTP(rec, req)

	if len(mr.Keys()) != 0 {
		t.Errorf("non-200 responses must not be cached, found keys: %v", mr.Keys())
	}
}

func TestInvalidateMiddlewareClearsOrgEntries(t *testing.T) {
	cm, mr := newTestCache(t)

	readCalls := 0
	router := gin.New()
	router.GET("/filaments", orgMiddleware(), cm.Cache5Min("filaments", func(c *gin.Context) {
		readCalls++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}))
	router.POST("/filaments", orgMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	}, cm.InvalidateMiddleware("filaments"))

	get := func(org string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/filaments", nil)
		req.Header.Set("X-Org", org)
		router.ServeHTTP(rec, req)
	}
	post := func(org string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/filaments", nil)
		req.Header.Set("X-Org", org)
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	get("org-a") // miss -> cached (readCalls=1)
	get("org-a") // hit (readCalls still 1)
	if readCalls != 1 {
		t.Fatalf("expected 1 read before invalidation, got %d", readCalls)
	}
	if len(mr.Keys()) != 1 {
		t.Fatalf("expected 1 cached entry, got %v", mr.Keys())
	}

	// A successful mutation must clear org-a's filament cache.
	if code := post("org-a"); code != http.StatusCreated {
		t.Fatalf("expected 201 from mutation, got %d", code)
	}
	if len(mr.Keys()) != 0 {
		t.Errorf("invalidation should clear org-a entries, remaining: %v", mr.Keys())
	}

	get("org-a") // miss again -> handler runs (readCalls=2)
	if readCalls != 2 {
		t.Errorf("expected read handler to run again after invalidation, got %d calls", readCalls)
	}
}

func TestInvalidatePrefixIsTenantScoped(t *testing.T) {
	cm, mr := newTestCache(t)

	router := gin.New()
	router.GET("/filaments", orgMiddleware(), cm.Cache5Min("filaments", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}))

	get := func(org string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/filaments", nil)
		req.Header.Set("X-Org", org)
		router.ServeHTTP(rec, req)
	}

	get("org-a")
	get("org-b")
	if len(mr.Keys()) != 2 {
		t.Fatalf("expected 2 entries before invalidation, got %v", mr.Keys())
	}

	// Invalidating org-a must not touch org-b's entry.
	if err := cm.InvalidatePrefix(context.Background(), "org-a", "filaments"); err != nil {
		t.Fatalf("invalidate failed: %v", err)
	}
	keys := mr.Keys()
	if len(keys) != 1 || !strings.Contains(keys[0], "org:org-b") {
		t.Errorf("only org-b entry should remain, got %v", keys)
	}
}
