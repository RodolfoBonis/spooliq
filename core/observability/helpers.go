package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Helper provides simplified observability helpers
type Helper struct {
	manager *ObservabilityManager
	logger  logger.Logger
}

// SpanOptions configure span creation
type SpanOptions struct {
	Component  string
	Operation  string
	Attributes []attribute.KeyValue
	Kind       trace.SpanKind
}

// MetricOptions configure metric recording
type MetricOptions struct {
	Component  string
	Attributes []attribute.KeyValue
}

// NewHelper creates a new observability helper
func NewHelper(manager *ObservabilityManager, logger logger.Logger) *Helper {
	return &Helper{
		manager: manager,
		logger:  logger,
	}
}

// === SPAN HELPERS ===

// StartSpan starts a new span with simplified configuration
func (h *Helper) StartSpan(ctx context.Context, name string, opts *SpanOptions) (context.Context, trace.Span) {
	if !h.manager.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	tracer := h.manager.GetTracer(component)
	spanOpts := []trace.SpanStartOption{}

	if opts != nil {
		if opts.Kind != trace.SpanKindUnspecified {
			spanOpts = append(spanOpts, trace.WithSpanKind(opts.Kind))
		}

		if len(opts.Attributes) > 0 {
			spanOpts = append(spanOpts, trace.WithAttributes(opts.Attributes...))
		}
	}

	ctx, span := tracer.Start(ctx, name, spanOpts...)

	// Add default attributes
	span.SetAttributes(
		attribute.String("component", component),
	)

	if opts != nil && opts.Operation != "" {
		span.SetAttributes(attribute.String("operation", opts.Operation))
	}

	return ctx, span
}

// TraceFunction automatically traces a function execution
func (h *Helper) TraceFunction(ctx context.Context, name string, fn func(context.Context) error, opts *SpanOptions) error {
	ctx, span := h.StartSpan(ctx, name, opts)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Record timing
	span.SetAttributes(
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	// Handle error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error(ctx, "Function execution failed", logger.Fields{
			"function":    name,
			"error":       err.Error(),
			"duration_ms": duration.Milliseconds(),
		})
	} else {
		span.SetStatus(codes.Ok, "")

		h.logger.Debug(ctx, "Function executed successfully", logger.Fields{
			"function":    name,
			"duration_ms": duration.Milliseconds(),
		})
	}

	return err
}

// TraceFunctionWithResult traces a function with return value
func TraceFunctionWithResult[T any](h *Helper, ctx context.Context, name string, fn func(context.Context) (T, error), opts *SpanOptions) (T, error) {
	ctx, span := h.StartSpan(ctx, name, opts)
	defer span.End()

	start := time.Now()
	result, err := fn(ctx)
	duration := time.Since(start)

	// Record timing
	span.SetAttributes(
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	// Handle error
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		h.logger.Error(ctx, "Function execution failed", logger.Fields{
			"function":    name,
			"error":       err.Error(),
			"duration_ms": duration.Milliseconds(),
		})
	} else {
		span.SetStatus(codes.Ok, "")

		h.logger.Debug(ctx, "Function executed successfully", logger.Fields{
			"function":    name,
			"duration_ms": duration.Milliseconds(),
		})
	}

	return result, err
}

// AddSpanEvent adds an event to the current span
func (h *Helper) AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	if !h.manager.IsEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// SetSpanAttributes sets attributes on the current span
func (h *Helper) SetSpanAttributes(ctx context.Context, attributes ...attribute.KeyValue) {
	if !h.manager.IsEnabled() {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetAttributes(attributes...)
	}
}

// RecordSpanError records an error on the current span
func (h *Helper) RecordSpanError(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	if !h.manager.IsEnabled() || err == nil {
		return
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.RecordError(err, trace.WithAttributes(attributes...))
		span.SetStatus(codes.Error, err.Error())
	}
}

// === METRIC HELPERS ===

// RecordDuration records a duration metric
func (h *Helper) RecordDuration(ctx context.Context, name string, duration time.Duration, opts *MetricOptions) {
	if !h.manager.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	meter := h.manager.GetMeter(component)
	histogram, err := meter.Float64Histogram(
		name,
		metric.WithDescription(fmt.Sprintf("Duration of %s operations", name)),
		metric.WithUnit("s"),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to create duration metric", logger.Fields{
			"metric": name,
			"error":  err.Error(),
		})
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// IncrementCounter increments a counter metric
func (h *Helper) IncrementCounter(ctx context.Context, name string, value int64, opts *MetricOptions) {
	if !h.manager.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	meter := h.manager.GetMeter(component)
	counter, err := meter.Int64Counter(
		name,
		metric.WithDescription(fmt.Sprintf("Counter for %s events", name)),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to create counter metric", logger.Fields{
			"metric": name,
			"error":  err.Error(),
		})
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// SetGauge sets a gauge metric value
func (h *Helper) SetGauge(ctx context.Context, name string, value int64, opts *MetricOptions) {
	if !h.manager.IsEnabled() {
		return
	}

	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	meter := h.manager.GetMeter(component)
	gauge, err := meter.Int64Gauge(
		name,
		metric.WithDescription(fmt.Sprintf("Gauge for %s values", name)),
	)
	if err != nil {
		h.logger.Error(ctx, "Failed to create gauge metric", logger.Fields{
			"metric": name,
			"error":  err.Error(),
		})
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("component", component),
	}
	if opts != nil && len(opts.Attributes) > 0 {
		attrs = append(attrs, opts.Attributes...)
	}

	gauge.Record(ctx, value, metric.WithAttributes(attrs...))
}

// === HIGH-LEVEL HELPERS ===

// TraceHTTPRequest traces an HTTP request
func (h *Helper) TraceHTTPRequest(ctx context.Context, method, url string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, fmt.Sprintf("HTTP %s %s", method, url), fn, &SpanOptions{
		Component: "http_client",
		Operation: "request",
		Kind:      trace.SpanKindClient,
		Attributes: []attribute.KeyValue{
			attribute.String("http.method", method),
			attribute.String("http.url", url),
		},
	})
}

// TraceDBQuery traces a database query
func (h *Helper) TraceDBQuery(ctx context.Context, query string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, "db.query", fn, &SpanOptions{
		Component: "database",
		Operation: "query",
		Kind:      trace.SpanKindClient,
		Attributes: []attribute.KeyValue{
			attribute.String("db.statement", query),
		},
	})
}

// TraceRedisOperation traces a Redis operation
func (h *Helper) TraceRedisOperation(ctx context.Context, operation, key string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, fmt.Sprintf("redis.%s", operation), fn, &SpanOptions{
		Component: "redis",
		Operation: operation,
		Kind:      trace.SpanKindClient,
		Attributes: []attribute.KeyValue{
			attribute.String("redis.operation", operation),
			attribute.String("redis.key", key),
		},
	})
}

// TraceAMQPPublish traces AMQP message publishing
func (h *Helper) TraceAMQPPublish(ctx context.Context, exchange, routingKey string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, fmt.Sprintf("amqp.publish %s", exchange), fn, &SpanOptions{
		Component: "amqp",
		Operation: "publish",
		Kind:      trace.SpanKindProducer,
		Attributes: []attribute.KeyValue{
			attribute.String("amqp.exchange", exchange),
			attribute.String("amqp.routing_key", routingKey),
		},
	})
}

// TraceAMQPConsume traces AMQP message consumption
func (h *Helper) TraceAMQPConsume(ctx context.Context, queue string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, fmt.Sprintf("amqp.consume %s", queue), fn, &SpanOptions{
		Component: "amqp",
		Operation: "consume",
		Kind:      trace.SpanKindConsumer,
		Attributes: []attribute.KeyValue{
			attribute.String("amqp.queue", queue),
		},
	})
}

// TraceBusinessOperation traces a business operation
func (h *Helper) TraceBusinessOperation(ctx context.Context, operation string, fn func(context.Context) error) error {
	return h.TraceFunction(ctx, operation, fn, &SpanOptions{
		Component: "business",
		Operation: operation,
		Kind:      trace.SpanKindInternal,
	})
}

// === BUSINESS METRIC HELPERS ===

// RecordUserAction records a user action metric
func (h *Helper) RecordUserAction(ctx context.Context, action, userID string) {
	h.IncrementCounter(ctx, "user_actions_total", 1, &MetricOptions{
		Component: "business",
		Attributes: []attribute.KeyValue{
			attribute.String("action", action),
			attribute.String("user_id", userID),
		},
	})
}

// RecordFeatureUsage records feature usage
func (h *Helper) RecordFeatureUsage(ctx context.Context, feature, userID string) {
	h.IncrementCounter(ctx, "feature_usage_total", 1, &MetricOptions{
		Component: "business",
		Attributes: []attribute.KeyValue{
			attribute.String("feature", feature),
			attribute.String("user_id", userID),
		},
	})
}

// RecordError records an error metric
func (h *Helper) RecordError(ctx context.Context, component, operation string, err error) {
	h.IncrementCounter(ctx, "errors_total", 1, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.String("operation", operation),
			attribute.String("error_type", fmt.Sprintf("%T", err)),
			attribute.String("error_message", err.Error()),
		},
	})
}

// RecordLatency records a latency metric
func (h *Helper) RecordLatency(ctx context.Context, operation string, duration time.Duration) {
	h.RecordDuration(ctx, fmt.Sprintf("%s_duration_seconds", operation), duration, &MetricOptions{
		Component: "performance",
		Attributes: []attribute.KeyValue{
			attribute.String("operation", operation),
		},
	})
}

// === COMPOSITE HELPERS ===

// TraceAndMeasure combines tracing and metrics for a function
func (h *Helper) TraceAndMeasure(ctx context.Context, name string, fn func(context.Context) error, opts *SpanOptions) error {
	start := time.Now()

	err := h.TraceFunction(ctx, name, fn, opts)
	duration := time.Since(start)

	// Record latency metric
	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	h.RecordDuration(ctx, fmt.Sprintf("%s_duration_seconds", name), duration, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.Bool("success", err == nil),
		},
	})

	// Record operation metric
	h.IncrementCounter(ctx, fmt.Sprintf("%s_operations_total", component), 1, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.String("operation", name),
			attribute.Bool("success", err == nil),
		},
	})

	// Record error if applicable
	if err != nil {
		h.RecordError(ctx, component, name, err)
	}

	return err
}

// TraceAndMeasureWithResult combines tracing and metrics for a function with result
func TraceAndMeasureWithResult[T any](h *Helper, ctx context.Context, name string, fn func(context.Context) (T, error), opts *SpanOptions) (T, error) {
	start := time.Now()

	result, err := TraceFunctionWithResult(h, ctx, name, fn, opts)
	duration := time.Since(start)

	// Record latency metric
	component := "default"
	if opts != nil && opts.Component != "" {
		component = opts.Component
	}

	h.RecordDuration(ctx, fmt.Sprintf("%s_duration_seconds", name), duration, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.Bool("success", err == nil),
		},
	})

	// Record operation metric
	h.IncrementCounter(ctx, fmt.Sprintf("%s_operations_total", component), 1, &MetricOptions{
		Component: component,
		Attributes: []attribute.KeyValue{
			attribute.String("operation", name),
			attribute.Bool("success", err == nil),
		},
	})

	// Record error if applicable
	if err != nil {
		h.RecordError(ctx, component, name, err)
	}

	return result, err
}

// === CONTEXT HELPERS ===

// GetTraceID extracts trace ID from context
func (h *Helper) GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts span ID from context
func (h *Helper) GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// IsTracing checks if tracing is active for the context
func (h *Helper) IsTracing(ctx context.Context) bool {
	if !h.manager.IsEnabled() {
		return false
	}

	span := trace.SpanFromContext(ctx)
	return span.SpanContext().IsValid()
}

// === ASYNC HELPERS ===

// TraceGoroutine traces a goroutine execution
func (h *Helper) TraceGoroutine(ctx context.Context, name string, fn func(context.Context)) {
	if !h.manager.IsEnabled() {
		go fn(ctx)
		return
	}

	ctx, span := h.StartSpan(ctx, name, &SpanOptions{
		Component: "async",
		Operation: "goroutine",
		Kind:      trace.SpanKindInternal,
	})

	go func() {
		defer span.End()

		start := time.Now()
		fn(ctx)
		duration := time.Since(start)

		span.SetAttributes(
			attribute.Int64("duration_ms", duration.Milliseconds()),
		)

		h.logger.Debug(ctx, "Goroutine completed", logger.Fields{
			"name":        name,
			"duration_ms": duration.Milliseconds(),
		})
	}()
}

// === UTILITY FUNCTIONS ===

// CreateCommonAttributes creates common attributes for a request
func (h *Helper) CreateCommonAttributes(userID, requestID string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	if userID != "" {
		attrs = append(attrs, attribute.String("user.id", userID))
	}

	if requestID != "" {
		attrs = append(attrs, attribute.String("request.id", requestID))
	}

	return attrs
}

// Global helper instance (will be set by the observability manager)
var GlobalHelper *Helper

// SetGlobalHelper sets the global helper instance
func SetGlobalHelper(helper *Helper) {
	GlobalHelper = helper
}

// === GLOBAL CONVENIENCE FUNCTIONS ===

// These functions provide global access to common observability operations

// Trace starts a new span
func Trace(ctx context.Context, name string, opts *SpanOptions) (context.Context, trace.Span) {
	if GlobalHelper != nil {
		return GlobalHelper.StartSpan(ctx, name, opts)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// Measure records a duration metric
func Measure(ctx context.Context, name string, duration time.Duration, opts *MetricOptions) {
	if GlobalHelper != nil {
		GlobalHelper.RecordDuration(ctx, name, duration, opts)
	}
}

// Count increments a counter metric
func Count(ctx context.Context, name string, value int64, opts *MetricOptions) {
	if GlobalHelper != nil {
		GlobalHelper.IncrementCounter(ctx, name, value, opts)
	}
}

// Event adds an event to the current span
func Event(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	if GlobalHelper != nil {
		GlobalHelper.AddSpanEvent(ctx, name, attributes...)
	}
}

// Error records an error on the current span
func Error(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	if GlobalHelper != nil {
		GlobalHelper.RecordSpanError(ctx, err, attributes...)
	}
}
