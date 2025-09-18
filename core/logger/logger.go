package logger

import (
	"context"
	"os"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Fields represents structured log fields.
type Fields map[string]interface{}

// Logger is a robust, structured logger interface for the whole system.
type Logger interface {
	Debug(ctx context.Context, message string, fields ...Fields)
	Info(ctx context.Context, message string, fields ...Fields)
	Warning(ctx context.Context, message string, fields ...Fields)
	Error(ctx context.Context, message string, fields ...Fields)
	Fatal(ctx context.Context, message string, fields ...Fields)
	Panic(ctx context.Context, message string, fields ...Fields)
	With(fields Fields) Logger
	LogError(ctx context.Context, message string, err error)
}

// CustomLogger is a zap-based implementation of Logger.
type CustomLogger struct {
	logger *zap.Logger
}

// LogData encapsula os dados do log.
type LogData struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Time    time.Time              `json:"time"`
	JSON    map[string]interface{} `json:"json,omitempty"`
}

// NewLogger creates a new robust logger instance for fx DI.
func NewLogger() Logger {
	var zapLogger *zap.Logger
	var cfg zap.Config
	if config.EnvironmentConfig() == entities.Environment.Development {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	// Dynamic log level from env
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		if level, err := zapcore.ParseLevel(lvl); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(level)
		}
	}
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapLogger, _ = cfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	return &CustomLogger{logger: zapLogger}
}

// Debug logs a debug message.
func (cl *CustomLogger) Debug(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Debug(message, cl.zapFields(ctx, fields...)...)
}

// Info logs an info message.
func (cl *CustomLogger) Info(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Info(message, cl.zapFields(ctx, fields...)...)
}

// Warning logs a warning message.
func (cl *CustomLogger) Warning(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Warn(message, cl.zapFields(ctx, fields...)...)
}

// Error logs an error message.
func (cl *CustomLogger) Error(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Error(message, cl.zapFields(ctx, fields...)...)
}

// Fatal logs a fatal message.
func (cl *CustomLogger) Fatal(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Fatal(message, cl.zapFields(ctx, fields...)...)
}

// Panic logs a panic message.
func (cl *CustomLogger) Panic(ctx context.Context, message string, fields ...Fields) {
	cl.logger.Panic(message, cl.zapFields(ctx, fields...)...)
}

// With returns a logger with additional fields.
func (cl *CustomLogger) With(fields Fields) Logger {
	return &CustomLogger{logger: cl.logger.With(cl.fieldsToZap(fields)...)}
}

// zapFields merges context and custom fields for structured logging.
func (cl *CustomLogger) zapFields(ctx context.Context, fields ...Fields) []zap.Field {
	var allFields = map[string]interface{}{}
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}
	// Add requestID from context if present
	if ctx != nil {
		if reqID, ok := ctx.Value("requestID").(string); ok && reqID != "" {
			allFields["requestID"] = reqID
		}
		if ip, ok := ctx.Value("ip").(string); ok && ip != "" {
			allFields["ip"] = ip
		}
	}
	return cl.fieldsToZap(allFields)
}

func (cl *CustomLogger) fieldsToZap(fields Fields) []zap.Field {
	zfs := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zfs = append(zfs, zap.Any(k, v))
	}
	return zfs
}

// Module provides the fx module for CustomLogger.
var Module = fx.Module("logger", fx.Provide(NewLogger))

// LogError logs any error in a structured way, extracting stacktrace/context if available.
func (cl *CustomLogger) LogError(ctx context.Context, message string, err error) {
	if err == nil {
		return
	}
	
	// Create a logger with additional caller skip for LogError method
	loggerWithSkip := cl.logger.WithOptions(zap.AddCallerSkip(0))
	
	var fields map[string]interface{}
	if appErr, ok := err.(interface{ ToLogFields() map[string]interface{} }); ok {
		fields = appErr.ToLogFields()
	} else {
		fields = map[string]interface{}{
			"error": err.Error(),
		}
	}
	
	// Add context fields to error fields
	zapFields := cl.zapFields(ctx, fields)
	loggerWithSkip.Error(message, zapFields...)
}
