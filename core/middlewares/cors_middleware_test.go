package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCorsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Cors())
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func preflight(router *gin.Engine, origin, method string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", method)
	router.ServeHTTP(rec, req)
	return rec
}

func TestCorsDefaultAllowsAllWithoutCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	router := newCorsRouter()

	rec := preflight(router, "https://anything.example.com", http.MethodPatch)

	allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected wildcard origin by default, got %q", allowOrigin)
	}
	if creds := rec.Header().Get("Access-Control-Allow-Credentials"); creds == "true" {
		t.Errorf("credentials must be disabled when allowing all origins, got %q", creds)
	}

	allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(allowMethods, m) {
			t.Errorf("expected method %s in Allow-Methods %q", m, allowMethods)
		}
	}
}

func TestCorsExplicitOriginsEnableCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	router := newCorsRouter()

	t.Run("allowed origin is echoed with credentials", func(t *testing.T) {
		rec := preflight(router, "https://app.example.com", http.MethodPatch)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
			t.Errorf("expected origin echoed, got %q", got)
		}
		if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
			t.Errorf("expected credentials enabled, got %q", got)
		}
	})

	t.Run("disallowed origin is rejected", func(t *testing.T) {
		rec := preflight(router, "https://evil.example.com", http.MethodGet)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "https://evil.example.com" || got == "*" {
			t.Errorf("disallowed origin must not be allowed, got %q", got)
		}
	})
}
