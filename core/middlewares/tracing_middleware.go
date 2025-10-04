package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware provides distributed tracing capabilities
type TracingMiddleware struct {
	logger      logger.Logger
	serviceName string
	enabled     bool
}

// NewTracingMiddleware creates a new tracing middleware instance
func NewTracingMiddleware(logger logger.Logger, serviceName string, enabled bool) *TracingMiddleware {
	return &TracingMiddleware{
		logger:      logger,
		serviceName: serviceName,
		enabled:     enabled,
	}
}

// Middleware returns the Gin middleware handler for tracing
func (t *TracingMiddleware) Middleware() gin.HandlerFunc {
	if !t.enabled {
		// Return a no-op middleware if tracing is disabled
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Use the otelgin middleware with custom configuration
	return otelgin.Middleware(t.serviceName,
		otelgin.WithTracerProvider(otel.GetTracerProvider()),
		otelgin.WithPropagators(otel.GetTextMapPropagator()),
		otelgin.WithSpanNameFormatter(t.spanNameFormatter),
		otelgin.WithFilter(t.filterEndpoint),
	)
}

// CustomTracing adds custom tracing logic on top of otelgin
func (t *TracingMiddleware) CustomTracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !t.enabled {
			c.Next()
			return
		}

		// Get the span from context
		span := trace.SpanFromContext(c.Request.Context())
		if span.SpanContext().IsValid() {
			// Add custom attributes to the span
			span.SetAttributes(
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("http.request_id", c.GetString("requestID")),
			)

			// Add user information if available
			if userID, exists := c.Get("user_id"); exists {
				span.SetAttributes(
					attribute.String("user.id", fmt.Sprintf("%v", userID)),
				)
			}

			if userRole, exists := c.Get("user_role"); exists {
				span.SetAttributes(
					attribute.String("user.role", fmt.Sprintf("%v", userRole)),
				)
			}

			// Set trace and span IDs in context for logger correlation
			traceID := span.SpanContext().TraceID().String()
			spanID := span.SpanContext().SpanID().String()

			c.Set("trace_id", traceID)
			c.Set("span_id", spanID)

			// Add trace ID to response header for debugging
			c.Header("X-Trace-Id", traceID)
		}

		c.Next()

		// After request processing, add response status to span
		if span.SpanContext().IsValid() {
			span.SetAttributes(
				attribute.Int("http.response.status_code", c.Writer.Status()),
				attribute.Int("http.response.size", c.Writer.Size()),
			)

			// Mark span as error if status code indicates an error
			if c.Writer.Status() >= 400 {
				span.SetAttributes(
					attribute.Bool("error", true),
				)
			}
		}
	}
}

// spanNameFormatter formats the span name
func (t *TracingMiddleware) spanNameFormatter(c *gin.Context) string {
	// Create more descriptive span names
	method := c.Request.Method
	path := c.Request.URL.Path

	// Use route pattern if available from gin context
	if route := c.FullPath(); route != "" {
		path = route
	}

	return fmt.Sprintf("%s %s", method, path)
}

// filterEndpoint determines which endpoints should be traced
func (t *TracingMiddleware) filterEndpoint(r *http.Request) bool {
	// Don't trace health check and metrics endpoints
	excludedPaths := []string{
		"/health_check",
		"/health",
		"/metrics",
		"/ready",
		"/live",
	}

	for _, path := range excludedPaths {
		if r.URL.Path == path {
			return false
		}
	}

	return true
}

// StartSpan starts a new span for manual instrumentation
func (t *TracingMiddleware) StartSpan(c *gin.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !t.enabled {
		return c.Request.Context(), trace.SpanFromContext(c.Request.Context())
	}

	tracer := otel.Tracer(t.serviceName)
	ctx, span := tracer.Start(c.Request.Context(), spanName, opts...)

	// Add common attributes
	span.SetAttributes(
		attribute.String("service.name", t.serviceName),
		attribute.String("request.id", c.GetString("requestID")),
	)

	return ctx, span
}

// InjectTraceContext injects trace context into outgoing requests
func (t *TracingMiddleware) InjectTraceContext(c *gin.Context, carrier propagation.TextMapCarrier) {
	if !t.enabled {
		return
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(c.Request.Context(), carrier)
}

// ExtractTraceContext extracts trace context from incoming requests
func (t *TracingMiddleware) ExtractTraceContext(c *gin.Context) context.Context {
	if !t.enabled {
		return c.Request.Context()
	}

	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
}

// NewTracingMiddlewareProvider creates a TracingMiddleware with FX dependencies
// DEPRECATED: Use observability.Instrumentor.InstrumentHTTPServer instead
// This function has been removed as TelemetryService has been replaced by ObservabilityManager
