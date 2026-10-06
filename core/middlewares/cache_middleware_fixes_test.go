package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestCacheBypassedWhenNoOrg verifies that a request without an organization_id in
// the context is never cached (no read, no write). A missing tenant previously
// fell back to a shared "none" namespace, which could serve one request's body to
// an unrelated caller.
func TestCacheBypassedWhenNoOrg(t *testing.T) {
	cm, mr := newTestCache(t)

	calls := 0
	router := gin.New()
	// No orgMiddleware here: organization_id is never set in the context.
	router.GET("/filaments", cm.Cache5Min("filaments", func(c *gin.Context) {
		calls++
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}))

	do := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/filaments", nil)
		router.ServeHTTP(rec, req)
		return rec
	}

	rec1 := do()
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	rec2 := do()

	// The handler must run on every request (no cache hit) and nothing is stored.
	if calls != 2 {
		t.Errorf("handler should run on every request when org is empty, got %d calls", calls)
	}
	if rec2.Header().Get("X-Cache") == "HIT" {
		t.Errorf("expected no cache HIT without an org, got %q", rec2.Header().Get("X-Cache"))
	}
	if len(mr.Keys()) != 0 {
		t.Errorf("no cache entries must be written without an org, found: %v", mr.Keys())
	}
}

// TestInvalidateMiddlewareNoOpWhenNoOrg verifies the invalidation middleware is a
// no-op (and never panics) when there is no organization in the context.
func TestInvalidateMiddlewareNoOpWhenNoOrg(t *testing.T) {
	cm, _ := newTestCache(t)

	router := gin.New()
	router.POST("/filaments", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	}, cm.InvalidateMiddleware("filaments"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/filaments", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

// TestCacheCapturesStringResponses verifies that c.String responses (which gin
// renders via WriteString, not Write) are captured and served from cache with the
// correct Content-Type. Before overriding WriteString, string bodies were cached
// empty and replayed as blank 200s.
func TestCacheCapturesStringResponses(t *testing.T) {
	cm, mr := newTestCache(t)

	const body = "hello world"
	calls := 0
	router := gin.New()
	router.GET("/text", orgMiddleware(), cm.Cache5Min("text", func(c *gin.Context) {
		calls++
		c.String(http.StatusOK, body)
	}))

	do := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/text", nil)
		req.Header.Set("X-Org", "org-a")
		router.ServeHTTP(rec, req)
		return rec
	}

	rec1 := do()
	if rec1.Code != http.StatusOK || rec1.Body.String() != body {
		t.Fatalf("miss: expected 200 %q, got %d %q", body, rec1.Code, rec1.Body.String())
	}
	if len(mr.Keys()) != 1 {
		t.Fatalf("string response should be cached, keys: %v", mr.Keys())
	}

	rec2 := do()
	if rec2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected HIT on second request, got %q", rec2.Header().Get("X-Cache"))
	}
	if calls != 1 {
		t.Errorf("handler should run once, got %d", calls)
	}
	if rec2.Body.String() != body {
		t.Errorf("cached string body mismatch: got %q want %q", rec2.Body.String(), body)
	}
	if ct := rec2.Header().Get("Content-Type"); ct == "" {
		t.Errorf("cache hit must restore a Content-Type, got empty")
	}
}

// TestCacheSkipsEmptyBody verifies a 200 with no body is not cached, so a later
// request still reaches the handler instead of replaying a blank entry.
func TestCacheSkipsEmptyBody(t *testing.T) {
	cm, mr := newTestCache(t)

	router := gin.New()
	router.GET("/empty", orgMiddleware(), cm.Cache5Min("empty", func(c *gin.Context) {
		c.Status(http.StatusOK) // 200, no body written
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	req.Header.Set("X-Org", "org-a")
	router.ServeHTTP(rec, req)

	if len(mr.Keys()) != 0 {
		t.Errorf("empty-body 200 must not be cached, found keys: %v", mr.Keys())
	}
}

// TestInvalidateMiddlewareMultiplePrefixes verifies a single mutation clears every
// listed resource prefix for the caller's org (e.g. a brand write also clearing
// filaments and dashboard), while leaving other orgs' entries untouched.
func TestInvalidateMiddlewareMultiplePrefixes(t *testing.T) {
	cm, mr := newTestCache(t)

	router := gin.New()
	mkGet := func(path, prefix string) {
		router.GET(path, orgMiddleware(), cm.Cache5Min(prefix, func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		}))
	}
	mkGet("/brands", "brands")
	mkGet("/filaments", "filaments")
	mkGet("/dashboard", "dashboard")

	router.POST("/brands", orgMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"created": true})
	}, cm.InvalidateMiddleware("brands", "filaments", "dashboard"))

	warm := func(path, org string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Org", org)
		router.ServeHTTP(rec, req)
	}

	// Warm org-a's brand, filament and dashboard caches, plus an org-b entry.
	warm("/brands", "org-a")
	warm("/filaments", "org-a")
	warm("/dashboard", "org-a")
	warm("/brands", "org-b")
	if len(mr.Keys()) != 4 {
		t.Fatalf("expected 4 warmed entries, got %v", mr.Keys())
	}

	// A successful brand mutation for org-a must clear all three org-a groups.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/brands", nil)
	req.Header.Set("X-Org", "org-a")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	keys := mr.Keys()
	if len(keys) != 1 {
		t.Fatalf("only org-b's entry should remain, got %v", keys)
	}
	if got := keys[0]; !strings.Contains(got, "org:org-b") || !strings.Contains(got, "brands") {
		t.Errorf("remaining key should be org-b's brand entry, got %q", got)
	}
}
