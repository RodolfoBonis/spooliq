package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
)

// TestCacheMiddlewareRedisUnavailable reproduces the production incident where
// Redis failed to initialize, leaving the client nil. Previously the cache read
// dereferenced the nil client and panicked, turning every cached endpoint
// (dashboard, brands, materials, filaments) into a 500. The cache must now fail
// open: the request reaches the handler and returns its response normally.
func TestCacheMiddlewareRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// NewRedisService without Init() => client is nil, exactly like a boot-time
	// Redis connection failure in production.
	redisService := services.NewRedisService(&logger.NoopLogger{}, nil)
	cm := NewCacheMiddleware(redisService, &logger.NoopLogger{})

	// gin.New() has no Recovery middleware, so a nil-pointer panic would
	// propagate and fail this test instead of being silently swallowed.
	router := gin.New()
	router.GET("/v1/brands", cm.Cache5Min(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"brands": []string{"acme"}})
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/brands", nil)

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 when Redis is unavailable, got %d (body=%s)",
			recorder.Code, recorder.Body.String())
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected handler response to reach the client on cache bypass, got empty body")
	}
}
