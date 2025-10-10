package middlewares

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/logger"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// MonitoringMiddleware provides monitoring capabilities for the application.
type MonitoringMiddleware struct {
	logger logger.Logger
}

// NewMonitoringMiddleware creates a new MonitoringMiddleware instance.
func NewMonitoringMiddleware(logger logger.Logger) *MonitoringMiddleware {
	return &MonitoringMiddleware{logger: logger}
}

// SentryMiddleware is a middleware for Sentry error tracking.
func (m *MonitoringMiddleware) SentryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{Repanic: true})
}

// LogMiddleware is a middleware for logging requests and responses.
func (m *MonitoringMiddleware) LogMiddleware(ctx *gin.Context) {
	requestID := uuid.NewString()
	ctx.Set("requestID", requestID)
	var responseBody = logger.HandleResponseBody(ctx.Writer)
	ctx.Writer = responseBody

	ctxWithIP := context.WithValue(ctx.Request.Context(), contextKey("ip"), ctx.ClientIP())
	ctx.Request = ctx.Request.WithContext(ctxWithIP)

	ctx.Next()
}
