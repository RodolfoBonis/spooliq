package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.uber.org/fx"
)

// TelemetryService handles OpenTelemetry integration for SigNoz
type TelemetryService struct {
	logger         logger.Logger
	tracerProvider *trace.TracerProvider
	enabled        bool
}

// TelemetryConfig holds telemetry configuration
type TelemetryConfig struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
	Environment string
	Version     string
}

// NewTelemetryService creates a new telemetry service
func NewTelemetryService(lc fx.Lifecycle, logger logger.Logger) (*TelemetryService, error) {
	cfg := loadTelemetryConfig()
	
	service := &TelemetryService{
		logger:  logger,
		enabled: cfg.Enabled,
	}

	if !cfg.Enabled {
		logger.Info(context.Background(), "Telemetry disabled, skipping initialization")
		return service, nil
	}

	// Initialize OpenTelemetry
	tp, err := initTracerProvider(cfg)
	if err != nil {
		logger.Error(context.Background(), "Failed to initialize tracer provider", logger.Fields{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	service.tracerProvider = tp

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info(ctx, "Telemetry service started", logger.Fields{
				"endpoint":     cfg.Endpoint,
				"service_name": cfg.ServiceName,
				"environment":  cfg.Environment,
			})
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return service.Shutdown(ctx)
		},
	})

	return service, nil
}

// loadTelemetryConfig loads telemetry configuration from environment
func loadTelemetryConfig() TelemetryConfig {
	enabled := os.Getenv("SIGNOZ_ENABLED") == "true" || 
		os.Getenv("OTEL_TRACES_ENABLED") == "true"
	
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "spooliq-api"
	}

	environment := config.EnvironmentConfig()
	if env := os.Getenv("ENV"); env != "" {
		environment = env
	}

	version := os.Getenv("VERSION")
	if version == "" {
		version = "1.0.0"
	}

	return TelemetryConfig{
		Enabled:     enabled,
		Endpoint:    endpoint,
		ServiceName: serviceName,
		Environment: environment,
		Version:     version,
	}
}

// initTracerProvider initializes the OpenTelemetry tracer provider
func initTracerProvider(cfg TelemetryConfig) (*trace.TracerProvider, error) {
	ctx := context.Background()

	// Create OTLP exporter
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(), // Use WithInsecure for non-TLS endpoints
		otlptracehttp.WithTimeout(10*time.Second),
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     10 * time.Second,
			MaxElapsedTime:  30 * time.Second,
		}),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			attribute.String("environment", cfg.Environment),
			attribute.String("deployment.environment", cfg.Environment),
			// K8s attributes (will be overridden by env vars if running in K8s)
			attribute.String("k8s.pod.name", os.Getenv("POD_NAME")),
			attribute.String("k8s.pod.ip", os.Getenv("POD_IP")),
			attribute.String("k8s.namespace.name", os.Getenv("POD_NAMESPACE")),
			attribute.String("k8s.node.name", os.Getenv("NODE_IP")),
		),
		resource.WithProcessPID(),
		resource.WithProcessExecutableName(),
		resource.WithProcessExecutablePath(),
		resource.WithProcessOwner(),
		resource.WithProcessRuntimeName(),
		resource.WithProcessRuntimeVersion(),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Sampling strategy based on environment
	var sampler trace.Sampler
	switch cfg.Environment {
	case "production":
		// Sample 10% of traces in production
		sampler = trace.TraceIDRatioBased(0.1)
	case "staging":
		// Sample 50% of traces in staging
		sampler = trace.TraceIDRatioBased(0.5)
	default:
		// Sample all traces in development
		sampler = trace.AlwaysSample()
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter,
			trace.WithBatchTimeout(5*time.Second),
			trace.WithMaxExportBatchSize(512),
			trace.WithMaxQueueSize(2048),
		),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// Shutdown gracefully shuts down the telemetry service
func (t *TelemetryService) Shutdown(ctx context.Context) error {
	if !t.enabled || t.tracerProvider == nil {
		return nil
	}

	t.logger.Info(ctx, "Shutting down telemetry service...")
	
	// Force flush any remaining spans
	if err := t.tracerProvider.ForceFlush(ctx); err != nil {
		t.logger.Error(ctx, "Error flushing tracer provider", logger.Fields{
			"error": err.Error(),
		})
	}

	// Shutdown the tracer provider
	if err := t.tracerProvider.Shutdown(ctx); err != nil {
		t.logger.Error(ctx, "Error shutting down tracer provider", logger.Fields{
			"error": err.Error(),
		})
		return err
	}

	t.logger.Info(ctx, "Telemetry service shutdown complete")
	return nil
}

// IsEnabled returns whether telemetry is enabled
func (t *TelemetryService) IsEnabled() bool {
	return t.enabled
}

// GetTracerProvider returns the tracer provider
func (t *TelemetryService) GetTracerProvider() *trace.TracerProvider {
	return t.tracerProvider
}