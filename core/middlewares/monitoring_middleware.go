package middlewares

import (
	"context"
	"net/http"
	"time"

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
	start := time.Now()
	requestID := uuid.NewString()
	ctx.Set("requestID", requestID)
	var responseBody = logger.HandleResponseBody(ctx.Writer)
	var requestBody = logger.
		HandleRequestBody(ctx.Request)
	ctx.Writer = responseBody

	// Adiciona o IP ao contexto do request
	ctxWithIP := context.WithValue(ctx.Request.Context(), contextKey("ip"), ctx.ClientIP())
	ctx.Request = ctx.Request.WithContext(ctxWithIP)

	m.logger.Info(ctx.Request.Context(), "Requisição iniciada", logger.Fields{
		"request_id":   requestID,
		"ip":           ctx.ClientIP(),
		"method":       ctx.Request.Method,
		"url":          ctx.Request.URL.String(),
		"user_agent":   ctx.Request.UserAgent(),
		"request_body": requestBody,
	})

	ctx.Next()

	latency := time.Since(start)
	status := ctx.Writer.Status()
	logFields := logger.Fields{
		"request_id":    requestID,
		"ip":            ctx.ClientIP(),
		"method":        ctx.Request.Method,
		"url":           ctx.Request.URL.String(),
		"user_agent":    ctx.Request.UserAgent(),
		"status":        status,
		"latency_ms":    latency.Milliseconds(),
		"request_body":  requestBody,
		"response_body": responseBody.Body.String(),
	}

	if isSuccessStatusCode(status) {
		m.logger.Info(ctx.Request.Context(), "Requisição concluída", logFields)
	} else {
		m.logger.Error(ctx.Request.Context(), "Requisição falhou", logFields)
	}
}

func isSuccessStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		return true
	default:
		return false
	}
}
