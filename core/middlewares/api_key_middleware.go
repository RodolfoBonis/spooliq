package middlewares

import (
	"context"

	keyGuardian "github.com/RodolfoBonis/go_key_guardian"
	"github.com/gin-gonic/gin"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
)

// APIKeyMiddleware provides API Key authentication middleware.
type APIKeyMiddleware struct {
	logger logger.Logger
	cfg    *config.AppConfig
}

// NewAPIKeyMiddleware creates a new API Key middleware instance.
func NewAPIKeyMiddleware(logger logger.Logger, cfg *config.AppConfig) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		logger: logger,
		cfg:    cfg,
	}
}

// ProtectWithAPIKey creates a middleware that validates API Key authentication.
func (m *APIKeyMiddleware) ProtectWithAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		
		// Get API Key from header
		apiKey := c.GetHeader("X-Api-Key")
		if apiKey == "" {
			// Try alternative header names
			if apiKey = c.GetHeader("X-API-Key"); apiKey == "" {
				if apiKey = c.GetHeader("Authorization"); apiKey != "" {
					// Handle Bearer token format for API keys
					if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
						apiKey = apiKey[7:]
					}
				}
			}
		}

		if len(apiKey) < 1 {
			appError := errors.MiddlewareError("API Key is required")
			httpError := appError.ToHTTPError()
			m.logger.LogError(ctx, "API Key authentication failed: missing API key", appError)
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		// Validate API Key using keyGuardian
		configs, err := keyGuardian.ValidateAPIKey(apiKey, m.cfg.ServiceID)
		if err != nil {
			appError := errors.MiddlewareError(err.Error())
			httpError := appError.ToHTTPError()
			m.logger.LogError(ctx, "API Key authentication failed: invalid key", appError)
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		// Log successful authentication
		m.logger.Info(ctx, "API Key authentication successful", map[string]interface{}{
			"application_id": configs.ID.String(),
			"service_id":     m.cfg.ServiceID,
		})

		// Set configs in context for use in handlers
		c.Set("api_key_configs", configs)
		c.Set("application_id", configs.ID.String())
		
		c.Next()
	}
}

// ProtectWithAPIKeyFunc is a standalone function for backward compatibility.
func ProtectWithAPIKeyFunc(logger logger.Logger, cfg *config.AppConfig) gin.HandlerFunc {
	middleware := NewAPIKeyMiddleware(logger, cfg)
	return middleware.ProtectWithAPIKey()
} 