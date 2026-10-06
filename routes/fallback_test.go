package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newEngineWithFallbacks() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	registerFallbacks(router)
	router.GET("/v1/known", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return router
}

func TestNoRouteEnvelope(t *testing.T) {
	router := newEngineWithFallbacks()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/v1/does-not-exist", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if env["code"] != "route_not_found" {
		t.Errorf("code = %v, want route_not_found", env["code"])
	}
	if env["error"] == nil || env["message"] == nil {
		t.Errorf("missing error/message: %v", env)
	}
}

func TestNoMethodEnvelope(t *testing.T) {
	router := newEngineWithFallbacks()
	w := httptest.NewRecorder()
	// /v1/known exists for GET; POST should yield 405.
	router.ServeHTTP(w, httptest.NewRequest("POST", "/v1/known", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
	var env map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if env["code"] != "method_not_allowed" {
		t.Errorf("code = %v, want method_not_allowed", env["code"])
	}
}
