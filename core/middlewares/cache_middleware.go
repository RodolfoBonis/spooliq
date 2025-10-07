package middlewares

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
)

// CacheConfig holds cache configuration for an endpoint.
type CacheConfig struct {
	TTL          time.Duration           // Time to live
	KeyPrefix    string                  // Prefix for cache keys
	VaryByUser   bool                    // Include user ID in cache key
	VaryByQuery  bool                    // Include query parameters in cache key
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

// Cache returns a middleware function that caches responses based on the provided configuration.
func (cm *CacheMiddleware) Cache(config CacheConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip caching for non-GET requests by default
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		// Check condition if provided
		if config.Condition != nil && !config.Condition(c) {
			c.Next()
			return
		}

		// Generate cache key
		cacheKey := cm.generateCacheKey(c, config)

		// Try to get cached response
		cachedResponse, appErr := cm.redisService.Get(c.Request.Context(), cacheKey)
		if appErr != nil {
			cm.logger.Error(c.Request.Context(), "Failed to get cached response", map[string]interface{}{
				"cache_key": cacheKey,
				"error":     appErr.Error(),
			})
			c.Next()
			return
		}

		// If cache hit, return cached response
		if cachedResponse != "" {
			var cachedData CachedResponse
			if err := cm.redisService.GetWithJSON(c.Request.Context(), cacheKey, &cachedData); err == nil {
				// Set headers
				for key, value := range cachedData.Headers {
					c.Header(key, value)
				}
				c.Header("X-Cache", "HIT")
				c.Data(cachedData.StatusCode, cachedData.ContentType, cachedData.Body)
				return
			}
		}

		// Cache miss - proceed with request and cache response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
			statusCode:     http.StatusOK,
			headers:        make(map[string]string),
		}
		c.Writer = writer

		c.Next()

		// Cache the response
		cachedData := CachedResponse{
			Body:        writer.body,
			StatusCode:  writer.statusCode,
			ContentType: writer.Header().Get("Content-Type"),
			Headers:     writer.headers,
		}

		if appErr := cm.redisService.SetWithJSON(c.Request.Context(), cacheKey, cachedData, config.TTL); appErr != nil {
			cm.logger.Error(c.Request.Context(), "Failed to cache response", map[string]interface{}{
				"cache_key": cacheKey,
				"error":     appErr.Error(),
			})
		} else {
			cm.logger.Debug(c.Request.Context(), "Response cached successfully", map[string]interface{}{
				"cache_key": cacheKey,
				"ttl":       config.TTL.String(),
			})
		}

		c.Header("X-Cache", "MISS")
	}
}

// CachedResponse represents a cached HTTP response.
type CachedResponse struct {
	Body        []byte            `json:"body"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers"`
}

// responseWriter wraps gin.ResponseWriter to capture response data.
type responseWriter struct {
	gin.ResponseWriter
	body       []byte
	statusCode int
	headers    map[string]string
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

// generateCacheKey creates a unique cache key based on the request and configuration.
func (cm *CacheMiddleware) generateCacheKey(c *gin.Context, config CacheConfig) string {
	var keyParts []string

	// Add prefix
	if config.KeyPrefix != "" {
		keyParts = append(keyParts, config.KeyPrefix)
	} else {
		keyParts = append(keyParts, "cache")
	}

	// Add path
	keyParts = append(keyParts, c.Request.URL.Path)

	// Add user ID if requested
	if config.VaryByUser {
		userID := cm.getUserID(c)
		if userID != "" {
			keyParts = append(keyParts, fmt.Sprintf("user:%s", userID))
		}
	}

	// Add query parameters if requested
	if config.VaryByQuery && len(c.Request.URL.RawQuery) > 0 {
		keyParts = append(keyParts, "query:"+c.Request.URL.RawQuery)
	}

	// Add specific headers if requested
	for _, headerName := range config.VaryByHeader {
		if headerValue := c.GetHeader(headerName); headerValue != "" {
			keyParts = append(keyParts, fmt.Sprintf("header:%s:%s", headerName, headerValue))
		}
	}

	// Create final key
	finalKey := strings.Join(keyParts, ":")

	// Hash the key if it's too long
	if len(finalKey) > 250 {
		hash := md5.Sum([]byte(finalKey))
		finalKey = config.KeyPrefix + ":" + hex.EncodeToString(hash[:])
	}

	return finalKey
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

	// Try user_uuid (alternative naming)
	if userUUID, exists := c.Get("user_uuid"); exists {
		if id, ok := userUUID.(string); ok {
			return id
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

// ClearCache removes cached responses for a specific pattern.
func (cm *CacheMiddleware) ClearCache(c *gin.Context, pattern string) error {
	// Note: This is a simple implementation. For production, you might want to use Redis SCAN
	// or maintain a separate index of cache keys for more efficient clearing.
	return cm.redisService.Delete(c.Request.Context(), pattern)
}

// Decorator functions for easy usage

// Cache5Min creates a cache middleware with 5 minutes TTL.
func (cm *CacheMiddleware) Cache5Min() gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL: 5 * time.Minute,
	})
}

// Cache15Min creates a cache middleware with 15 minutes TTL.
func (cm *CacheMiddleware) Cache15Min() gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL: 15 * time.Minute,
	})
}

// Cache1Hour creates a cache middleware with 1 hour TTL.
func (cm *CacheMiddleware) Cache1Hour() gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL: 1 * time.Hour,
	})
}

// CacheUserSpecific creates a cache middleware that varies by user.
func (cm *CacheMiddleware) CacheUserSpecific(ttl time.Duration) gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL:        ttl,
		VaryByUser: true,
	})
}

// CacheWithQuery creates a cache middleware that includes query parameters.
func (cm *CacheMiddleware) CacheWithQuery(ttl time.Duration) gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL:         ttl,
		VaryByQuery: true,
	})
}

// CacheConditional creates a conditional cache middleware.
func (cm *CacheMiddleware) CacheConditional(ttl time.Duration, condition func(*gin.Context) bool) gin.HandlerFunc {
	return cm.Cache(CacheConfig{
		TTL:       ttl,
		Condition: condition,
	})
}
