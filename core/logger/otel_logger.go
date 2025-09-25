package logger

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// TraceLogger extends the Logger interface with trace-aware logging methods
type TraceLogger interface {
	Logger
	DebugWithTrace(ctx context.Context, message string, fields ...Fields)
	InfoWithTrace(ctx context.Context, message string, fields ...Fields)
	WarningWithTrace(ctx context.Context, message string, fields ...Fields)
	ErrorWithTrace(ctx context.Context, message string, fields ...Fields)
}

// withTraceContext enriches log fields with trace context if available
func withTraceContext(ctx context.Context, fields ...Fields) Fields {
	// Get span from context
	span := trace.SpanFromContext(ctx)

	// Create base fields map
	var baseFields Fields
	if len(fields) > 0 {
		baseFields = fields[0]
	} else {
		baseFields = Fields{}
	}

	// Add trace context if span is valid
	if span.SpanContext().IsValid() {
		baseFields["trace_id"] = span.SpanContext().TraceID().String()
		baseFields["span_id"] = span.SpanContext().SpanID().String()
		baseFields["trace_flags"] = span.SpanContext().TraceFlags().String()

		// Add trace state if present
		if span.SpanContext().TraceState().String() != "" {
			baseFields["trace_state"] = span.SpanContext().TraceState().String()
		}
	}

	return baseFields
}

// DebugWithTrace logs a debug message with trace context
func (cl *CustomLogger) DebugWithTrace(ctx context.Context, message string, fields ...Fields) {
	enrichedFields := withTraceContext(ctx, fields...)
	cl.Debug(ctx, message, enrichedFields)
}

// InfoWithTrace logs an info message with trace context
func (cl *CustomLogger) InfoWithTrace(ctx context.Context, message string, fields ...Fields) {
	enrichedFields := withTraceContext(ctx, fields...)
	cl.Info(ctx, message, enrichedFields)
}

// WarningWithTrace logs a warning message with trace context
func (cl *CustomLogger) WarningWithTrace(ctx context.Context, message string, fields ...Fields) {
	enrichedFields := withTraceContext(ctx, fields...)
	cl.Warning(ctx, message, enrichedFields)
}

// ErrorWithTrace logs an error message with trace context
func (cl *CustomLogger) ErrorWithTrace(ctx context.Context, message string, fields ...Fields) {
	enrichedFields := withTraceContext(ctx, fields...)
	cl.Error(ctx, message, enrichedFields)
}

// LogErrorWithTrace logs an error with trace context
func (cl *CustomLogger) LogErrorWithTrace(ctx context.Context, message string, err error) {
	if err == nil {
		return
	}

	// Get trace context
	span := trace.SpanFromContext(ctx)

	// Create fields from error
	var fields map[string]interface{}
	if appErr, ok := err.(interface{ ToLogFields() map[string]interface{} }); ok {
		fields = appErr.ToLogFields()
	} else {
		fields = map[string]interface{}{
			"error": err.Error(),
		}
	}

	// Add trace context if available
	if span.SpanContext().IsValid() {
		fields["trace_id"] = span.SpanContext().TraceID().String()
		fields["span_id"] = span.SpanContext().SpanID().String()

		// Record error in span
		span.RecordError(err)
	}

	// Log with enriched fields
	cl.Error(ctx, message, fields)
}

// NewTraceLogger creates a new trace-aware logger
func NewTraceLogger() TraceLogger {
	return NewLogger().(*CustomLogger)
}

// AddTraceToContext adds trace information to the context fields
func AddTraceToContext(ctx context.Context) Fields {
	fields := Fields{}

	// Check if we have trace context from Gin context
	if ginCtx, ok := ctx.Value("gin_context").(*gin.Context); ok {
		if traceID, exists := ginCtx.Get("trace_id"); exists {
			fields["trace_id"] = traceID
		}
		if spanID, exists := ginCtx.Get("span_id"); exists {
			fields["span_id"] = spanID
		}
		return fields
	}

	// Otherwise, get from OpenTelemetry span
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		fields["trace_id"] = span.SpanContext().TraceID().String()
		fields["span_id"] = span.SpanContext().SpanID().String()
	}

	return fields
}

// StartSpanWithLogger starts a new span and logs it
func StartSpanWithLogger(ctx context.Context, tracer trace.Tracer, spanName string, logger Logger) (context.Context, trace.Span) {
	ctx, span := tracer.Start(ctx, spanName)

	if span.SpanContext().IsValid() {
		logger.Debug(ctx, "Span started", Fields{
			"span_name": spanName,
			"trace_id":  span.SpanContext().TraceID().String(),
			"span_id":   span.SpanContext().SpanID().String(),
		})
	}

	return ctx, span
}

// EndSpanWithLogger ends a span and logs it
func EndSpanWithLogger(span trace.Span, logger Logger, err error) {
	if err != nil {
		span.RecordError(err)
		logger.Error(context.Background(), "Span ended with error", Fields{
			"span_id": span.SpanContext().SpanID().String(),
			"error":   err.Error(),
		})
	} else {
		logger.Debug(context.Background(), "Span ended successfully", Fields{
			"span_id": span.SpanContext().SpanID().String(),
		})
	}
	span.End()
}
