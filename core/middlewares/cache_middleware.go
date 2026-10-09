package middlewares

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
)

// CacheConfig holds cache configuration for an endpoint.
type CacheConfig struct {
	TTL          time.Duration           // Time to live
	KeyPrefix    string                  // Prefix for cache keys (also the invalidation group, e.g. "filaments")
	VaryByUser   bool                    // Include user ID in cache key (org ID and query are ALWAYS included)
	VaryByHeader []string                // Include specific headers in cache key
	Condition    func(*gin.Context) bool // Optional condition to enable cache
}

// CacheMiddleware provides caching functionality for HTTP endpoints.
type CacheMiddleware struct {
	redisService *services.RedisService
	logger       logger.Logger
}

// NewCacheMiddleware creates a new cache middleware instance.
func NewCacheMiddleware(redisService *services.RedisService, logger logger.Logger) *CacheMiddleware {
	return &CacheMiddleware{
		redisService: redisService,
		logger:       logger,
	}
}

// Wrap decorates a business handler with read-through caching.
//
// Why wrap the handler instead of running as a trailing middleware:
// protectFactory (see auth_middleware.go) runs authentication and then calls the
// handler INLINE without c.Next(). A cache registered after it in the route chain
// therefore executes AFTER the response is already written — too late to serve a
// cache hit. Wrapping makes the cache run *inside* protectFactory's handler slot,
// i.e. after auth (so organization_id is in the context) but before the real
// handler, which is exactly where caching must happen.
//
// Usage: protectFactory(cacheMiddleware.Wrap(useCase.FindAll, cfg), roles...)
func (cm *CacheMiddleware) Wrap(handler gin.HandlerFunc, config CacheConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		cm.serve(c, config, handler)
	}
}

// serve contains the shared read-through cache logic. `next` is the function that
// produces the real response on a cache miss.
func (cm *CacheMiddleware) serve(c *gin.Context, config CacheConfig, next gin.HandlerFunc) {
	// Only GET requests are cacheable.
	if c.Request.Method != http.MethodGet {
		next(c)
		return
	}

	// Without an organization ID we cannot build a tenant-isolated key. Bypass the
	// cache entirely (no read, no write) rather than falling back to a shared
	// namespace, which could leak one tenant's response to another.
	if cm.getOrganizationID(c) == "" {
		next(c)
		return
	}

	// Optional gate.
	if config.Condition != nil && !config.Condition(c) {
		next(c)
		return
	}

	cacheKey := cm.generateCacheKey(c, config)

	// Try to serve from cache. redisService fails open (returns empty on any
	// error or when Redis is unavailable), so this never blocks the request.
	var cachedData CachedResponse
	if appErr := cm.redisService.GetWithJSON(c.Request.Context(), cacheKey, &cachedData); appErr != nil {
		cm.logger.Error(c.Request.Context(), "Failed to read cached response", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     appErr.Error(),
		})
	} else if cachedData.StatusCode != 0 {
		c.Header("X-Cache", "HIT")
		// c.Data sets the Content-Type header and writes the body, so the cached
		// Content-Type is restored exactly as produced on the original miss.
		c.Data(cachedData.StatusCode, cachedData.ContentType, cachedData.Body)
		c.Abort() // prevent any later handler from re-writing the response
		return
	}

	// Cache miss: capture the response while streaming it to the client.
	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           make([]byte, 0),
		statusCode:     http.StatusOK,
	}
	c.Writer = writer
	c.Header("X-Cache", "MISS")

	next(c)

	// Only cache successful 200 responses that actually produced a body. A 200
	// with an empty body (e.g. a handler that failed to write via a captured
	// path) must not poison the cache with a blank entry.
	if writer.statusCode != http.StatusOK || len(writer.body) == 0 {
		return
	}
	// Handlers can opt a response out (e.g. a degraded, partial result).
	if strings.Contains(writer.Header().Get("Cache-Control"), "no-store") {
		return
	}

	cachedData = CachedResponse{
		Body:        writer.body,
		StatusCode:  writer.statusCode,
		ContentType: writer.Header().Get("Content-Type"),
	}

	if appErr := cm.redisService.SetWithJSON(c.Request.Context(), cacheKey, cachedData, config.TTL); appErr != nil {
		cm.logger.Error(c.Request.Context(), "Failed to cache response", map[string]interface{}{
			"cache_key": cacheKey,
			"error":     appErr.Error(),
		})
	}
}

// CachedResponse represents a cached HTTP response.
type CachedResponse struct {
	Body        []byte `json:"body"`
	StatusCode  int    `json:"status_code"`
	ContentType string `json:"content_type"`
}

// responseWriter wraps gin.ResponseWriter to capture the response body and status
// code while still streaming them to the client. Both Write and WriteString are
// overridden because gin renderers use either path (c.JSON uses Write, c.String
// uses WriteString); capturing only Write would silently cache an empty body for
// string/text responses.
type responseWriter struct {
	gin.ResponseWriter
	body       []byte
	statusCode int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, s...)
	return w.ResponseWriter.WriteString(s)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// cacheRoot is the global namespace for all response-cache keys.
const cacheRoot = "cache"

// stableKeyPrefix returns the deterministic prefix shared by every cache entry of
// a given resource within a given organization. It is both the start of each
// cache key and the match prefix used for invalidation, guaranteeing that a SCAN
// over "<stable>:*" only ever touches a single org's entries for a single
// resource — org A can never read or clear org B's cache.
//
// Callers must never pass an empty orgID: serve() bypasses caching and
// InvalidatePrefix no-ops before reaching here, so there is no "none" tenant.
func stableKeyPrefix(prefix, orgID string) string {
	if prefix == "" {
		prefix = cacheRoot
	}
	return fmt.Sprintf("%s:%s:org:%s", cacheRoot, prefix, orgID)
}

// generateCacheKey creates a unique cache key for the request.
//
// The key ALWAYS includes the organization ID and the full query string, and the
// user ID when VaryByUser is set. The organization ID is a hard requirement for
// tenant isolation and is not configurable.
func (cm *CacheMiddleware) generateCacheKey(c *gin.Context, config CacheConfig) string {
	stable := stableKeyPrefix(config.KeyPrefix, cm.getOrganizationID(c))

	variableParts := []string{"path:" + c.Request.URL.Path}

	if raw := c.Request.URL.RawQuery; raw != "" {
		variableParts = append(variableParts, "query:"+raw)
	}

	if config.VaryByUser {
		if userID := cm.getUserID(c); userID != "" {
			variableParts = append(variableParts, "user:"+userID)
		}
	}

	for _, headerName := range config.VaryByHeader {
		if headerValue := c.GetHeader(headerName); headerValue != "" {
			variableParts = append(variableParts, fmt.Sprintf("header:%s:%s", headerName, headerValue))
		}
	}

	variable := strings.Join(variableParts, ":")
	finalKey := stable + ":" + variable

	// Keep keys bounded, but preserve the stable prefix so invalidation still
	// matches hashed keys.
	if len(finalKey) > 250 {
		hash := md5.Sum([]byte(variable))
		finalKey = stable + ":" + hex.EncodeToString(hash[:])
	}

	return finalKey
}

// getOrganizationID extracts the tenant identifier set by the auth middleware.
func (cm *CacheMiddleware) getOrganizationID(c *gin.Context) string {
	if orgID, exists := c.Get("organization_id"); exists {
		if id, ok := orgID.(string); ok {
			return id
		}
	}
	return ""
}

// getUserID attempts to get user ID from various sources in the context.
func (cm *CacheMiddleware) getUserID(c *gin.Context) string {
	// Try direct user_id first
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}

	// Try from claims
	if claimsInterface, exists := c.Get("claims"); exists {
		if claims, ok := claimsInterface.(map[string]interface{}); ok {
			// Handle UUID type
			if id, ok := claims["ID"].(fmt.Stringer); ok {
				return id.String()
			}
			// Handle string type
			if id, ok := claims["ID"].(string); ok {
				return id
			}
		}
	}

	// Try sub claim (JWT standard)
	if sub, exists := c.Get("sub"); exists {
		if id, ok := sub.(string); ok {
			return id
		}
	}

	return ""
}

// InvalidateMiddleware returns a trailing middleware for mutating routes
// (POST/PUT/DELETE). After the handler runs, if the response was successful
// (2xx), it clears every cached entry for the given resource prefixes within the
// caller's organization. Placed after protectFactory so organization_id is
// already in the context and so it runs after the handler has written its status.
//
// Multiple prefixes let a single mutation invalidate related caches: e.g. a brand
// update must also clear "filaments" (filament responses embed brand names) and
// "dashboard" (aggregations depend on brands).
func (cm *CacheMiddleware) InvalidateMiddleware(prefixes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		status := c.Writer.Status()
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			return
		}

		orgID := cm.getOrganizationID(c)
		// Without a tenant we never wrote any cache entry, so there is nothing to
		// invalidate; skip to avoid touching a shared/foreign namespace.
		if orgID == "" {
			return
		}

		for _, prefix := range prefixes {
			if err := cm.InvalidatePrefix(c.Request.Context(), orgID, prefix); err != nil {
				cm.logger.Error(c.Request.Context(), "Failed to invalidate cache", map[string]interface{}{
					"prefix":          prefix,
					"organization_id": orgID,
					"error":           err.Error(),
				})
			}
		}
	}
}

// InvalidatePrefix removes every cached entry for a resource prefix scoped to a
// single organization, using a non-blocking SCAN + DEL (never KEYS). It fails
// open: when Redis is unavailable the call is a no-op and returns nil so
// mutations are never blocked by cache maintenance. An empty orgID is a no-op
// since serve() never writes entries for a missing tenant.
func (cm *CacheMiddleware) InvalidatePrefix(ctx context.Context, orgID, prefix string) error {
	if orgID == "" {
		return nil
	}

	client := cm.redisService.GetClient()
	if client == nil {
		return nil // fail-open: cache unavailable
	}

	match := stableKeyPrefix(prefix, orgID) + ":*"

	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, match, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Decorator functions for easy usage. Each takes the invalidation/cache prefix
// (the resource name, e.g. "filaments") and the handler to wrap.

// Cache5Min wraps a handler with a 5 minute TTL cache.
func (cm *CacheMiddleware) Cache5Min(prefix string, handler gin.HandlerFunc) gin.HandlerFunc {
	return cm.Wrap(handler, CacheConfig{TTL: 5 * time.Minute, KeyPrefix: prefix})
}

// Cache15Min wraps a handler with a 15 minute TTL cache.
func (cm *CacheMiddleware) Cache15Min(prefix string, handler gin.HandlerFunc) gin.HandlerFunc {
	return cm.Wrap(handler, CacheConfig{TTL: 15 * time.Minute, KeyPrefix: prefix})
}

// Cache1Hour wraps a handler with a 1 hour TTL cache.
func (cm *CacheMiddleware) Cache1Hour(prefix string, handler gin.HandlerFunc) gin.HandlerFunc {
	return cm.Wrap(handler, CacheConfig{TTL: 1 * time.Hour, KeyPrefix: prefix})
}
