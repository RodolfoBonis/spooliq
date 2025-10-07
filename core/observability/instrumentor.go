package observability

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

// Instrumentor provides automatic instrumentation capabilities
type Instrumentor struct {
	manager *Manager
	logger  logger.Logger
	config  *Config

	// Metrics
	httpDuration   metric.Float64Histogram
	dbDuration     metric.Float64Histogram
	redisDuration  metric.Float64Histogram
	amqpDuration   metric.Float64Histogram
	errorCounter   metric.Int64Counter
	requestCounter metric.Int64Counter
}

// InstrumentationContext holds context for instrumentation
type InstrumentationContext struct {
	SpanName   string
	Component  string
	Operation  string
	Attributes []attribute.KeyValue
	StartTime  time.Time
}

// NewInstrumentor creates a new instrumentor
func NewInstrumentor(manager *Manager, logger logger.Logger) (*Instrumentor, error) {
	inst := &Instrumentor{
		manager: manager,
		logger:  logger,
		config:  manager.GetConfig(),
	}

	if manager.IsEnabled() {
		if err := inst.initMetrics(); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	return inst, nil
}

// initMetrics initializes instrumentation metrics
func (i *Instrumentor) initMetrics() error {
	meter := i.manager.GetMeter("instrumentation")

	var err error

	// HTTP duration histogram
	i.httpDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(i.config.Metrics.HTTPLatencyBoundaries...),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP duration metric: %w", err)
	}

	// Database duration histogram
	i.dbDuration, err = meter.Float64Histogram(
		"db_query_duration_seconds",
		metric.WithDescription("Database query duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(i.config.Metrics.DBLatencyBoundaries...),
	)
	if err != nil {
		return fmt.Errorf("failed to create DB duration metric: %w", err)
	}

	// Redis duration histogram
	i.redisDuration, err = meter.Float64Histogram(
		"redis_operation_duration_seconds",
		metric.WithDescription("Redis operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create Redis duration metric: %w", err)
	}

	// AMQP duration histogram
	i.amqpDuration, err = meter.Float64Histogram(
		"amqp_operation_duration_seconds",
		metric.WithDescription("AMQP operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return fmt.Errorf("failed to create AMQP duration metric: %w", err)
	}

	// Error counter
	i.errorCounter, err = meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return fmt.Errorf("failed to create error counter: %w", err)
	}

	// Request counter
	i.requestCounter, err = meter.Int64Counter(
		"requests_total",
		metric.WithDescription("Total number of requests"),
	)
	if err != nil {
		return fmt.Errorf("failed to create request counter: %w", err)
	}

	return nil
}

// InstrumentHTTPServer creates an instrumented Gin middleware
func (i *Instrumentor) InstrumentHTTPServer(serviceName string) gin.HandlerFunc {
	if !i.config.Features.AutoHTTP {
		return func(c *gin.Context) { c.Next() }
	}

	return gin.HandlerFunc(func(c *gin.Context) {
		// Skip instrumentation for excluded paths
		for _, path := range i.config.Traces.ExcludedPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		start := time.Now()

		// Create span manually using the global tracer
		tracer := otel.Tracer(serviceName)
		
		// DEBUG: Log span creation
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		fmt.Printf("[OTEL DEBUG] Creating span: %s (path=%s, fullPath=%s)\n", 
			spanName, c.Request.URL.Path, c.FullPath())
		
		ctx, span := tracer.Start(c.Request.Context(), spanName)
		defer span.End()

		// Update the request context with the span
		c.Request = c.Request.WithContext(ctx)

		// Add HTTP attributes to span
		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.scheme", c.Request.URL.Scheme),
			attribute.String("http.host", c.Request.Host),
			attribute.String("http.target", c.Request.URL.Path),
			attribute.String("http.route", c.FullPath()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
		)

		// Continue with the request
		c.Next()

		// Add response attributes
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// Set span status based on HTTP status
		if c.Writer.Status() >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
			if c.Writer.Status() >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
			} else {
				span.SetStatus(codes.Ok, "")
			}
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Add custom instrumentation
		i.instrumentHTTPRequest(c, start)
	})
}

// instrumentHTTPRequest adds custom HTTP instrumentation
func (i *Instrumentor) instrumentHTTPRequest(c *gin.Context, start time.Time) {
	duration := time.Since(start).Seconds()

	// Get span and add attributes
	span := trace.SpanFromContext(c.Request.Context())
	if span.SpanContext().IsValid() {
		// Add custom attributes
		span.SetAttributes(
			attribute.String("http.client_ip", c.ClientIP()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.Int("http.request_content_length", int(c.Request.ContentLength)),
			attribute.Int("http.response_content_length", c.Writer.Size()),
		)

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", userID)))
		}
		if userRole, exists := c.Get("user_role"); exists {
			span.SetAttributes(attribute.String("user.role", fmt.Sprintf("%v", userRole)))
		}

		// Mark as error if status >= 400
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", c.Writer.Status()))
		}
	}

	// Record metrics
	if i.httpDuration != nil {
		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("route", c.FullPath()),
			attribute.Int("status_code", c.Writer.Status()),
		}
		i.httpDuration.Record(c.Request.Context(), duration, metric.WithAttributes(attrs...))
	}

	if i.requestCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("method", c.Request.Method),
			attribute.String("route", c.FullPath()),
			attribute.Int("status_code", c.Writer.Status()),
		}
		i.requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
	}

	// Record errors
	if c.Writer.Status() >= 400 && i.errorCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("component", "http"),
			attribute.String("method", c.Request.Method),
			attribute.String("route", c.FullPath()),
			attribute.Int("status_code", c.Writer.Status()),
		}
		i.errorCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
	}
}

// InstrumentHTTPClient creates an instrumented HTTP client
func (i *Instrumentor) InstrumentHTTPClient(client *http.Client) *http.Client {
	if !i.config.Features.AutoHTTP || client == nil {
		if client == nil {
			client = &http.Client{}
		}
		return client
	}

	// Wrap transport with OpenTelemetry instrumentation
	client.Transport = otelhttp.NewTransport(
		client.Transport,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("HTTP %s %s", r.Method, r.URL.Host)
		}),
	)

	return client
}

// InstrumentDatabase adds database instrumentation to GORM
func (i *Instrumentor) InstrumentDatabase(db *gorm.DB) error {
	if !i.config.Features.AutoDatabase {
		return nil
	}

	// Use GORM OpenTelemetry plugin
	if err := db.Use(tracing.NewPlugin()); err != nil {
		return fmt.Errorf("failed to add GORM OpenTelemetry plugin: %w", err)
	}

	// Add custom callback for metrics
	db.Callback().Query().After("gorm:query").Register("otel:query_metrics", i.dbMetricsCallback)
	db.Callback().Create().After("gorm:create").Register("otel:create_metrics", i.dbMetricsCallback)
	db.Callback().Update().After("gorm:update").Register("otel:update_metrics", i.dbMetricsCallback)
	db.Callback().Delete().After("gorm:delete").Register("otel:delete_metrics", i.dbMetricsCallback)

	return nil
}

// dbMetricsCallback records database metrics
func (i *Instrumentor) dbMetricsCallback(db *gorm.DB) {
	if i.dbDuration == nil {
		return
	}

	// Get elapsed time from GORM context
	duration := time.Since(db.Statement.Context.Value("start_time").(time.Time)).Seconds()

	attrs := []attribute.KeyValue{
		attribute.String("db.operation", db.Statement.SQL.String()),
		attribute.String("db.table", db.Statement.Table),
	}

	i.dbDuration.Record(db.Statement.Context, duration, metric.WithAttributes(attrs...))

	// Record errors
	if db.Error != nil && i.errorCounter != nil {
		errorAttrs := []attribute.KeyValue{
			attribute.String("component", "database"),
			attribute.String("db.table", db.Statement.Table),
			attribute.String("error", db.Error.Error()),
		}
		i.errorCounter.Add(db.Statement.Context, 1, metric.WithAttributes(errorAttrs...))
	}
}

// TraceFunction automatically instruments a function with tracing
func (i *Instrumentor) TraceFunction(ctx context.Context, fn interface{}, args ...interface{}) ([]interface{}, error) {
	if !i.manager.IsEnabled() {
		// Call function directly without instrumentation
		return i.callFunction(fn, args...)
	}

	// Get function name for span
	fnValue := reflect.ValueOf(fn)
	fnName := runtime.FuncForPC(fnValue.Pointer()).Name()

	tracer := i.manager.GetTracer("auto-instrumentation")
	ctx, span := tracer.Start(ctx, fnName)
	defer span.End()

	// Add function attributes
	span.SetAttributes(
		attribute.String("function.name", fnName),
		attribute.Int("function.args_count", len(args)),
	)

	start := time.Now()
	results, err := i.callFunction(fn, args...)
	duration := time.Since(start)

	// Record metrics and error handling
	span.SetAttributes(attribute.Float64("function.duration_ms", float64(duration.Nanoseconds())/1e6))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		if i.errorCounter != nil {
			attrs := []attribute.KeyValue{
				attribute.String("component", "function"),
				attribute.String("function.name", fnName),
			}
			i.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		}
	}

	return results, err
}

// callFunction uses reflection to call a function
func (i *Instrumentor) callFunction(fn interface{}, args ...interface{}) ([]interface{}, error) {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return nil, fmt.Errorf("provided value is not a function")
	}

	// Convert args to reflect.Value
	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		reflectArgs[i] = reflect.ValueOf(arg)
	}

	// Call function
	results := fnValue.Call(reflectArgs)

	// Convert results back
	interfaceResults := make([]interface{}, len(results))
	var err error

	for i, result := range results {
		interfaceResults[i] = result.Interface()

		// Check if last result is an error
		if i == len(results)-1 && result.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				err = result.Interface().(error)
			}
		}
	}

	return interfaceResults, err
}

// InjectContext injects trace context into outgoing requests
func (i *Instrumentor) InjectContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	if !i.manager.IsEnabled() {
		return
	}

	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, carrier)
}

// ExtractContext extracts trace context from incoming requests
func (i *Instrumentor) ExtractContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if !i.manager.IsEnabled() {
		return ctx
	}

	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, carrier)
}

// StartSpan starts a new span with automatic attribute enrichment
func (i *Instrumentor) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !i.manager.IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}

	tracer := i.manager.GetTracer("manual-instrumentation")
	return tracer.Start(ctx, name, opts...)
}
