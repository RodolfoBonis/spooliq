package observability

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otlpmetric "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otlptrace "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/fx"
)

// Manager manages all observability components
type Manager struct {
	config   *Config
	logger   logger.Logger
	resource *resource.Resource

	// Core providers
	tracerProvider *TraceProvider
	meterProvider  *MetricProvider
	loggerProvider *LogProvider

	// Component managers
	instrumentor *Instrumentor
	collector    *MetricCollector

	// State management
	mu        sync.RWMutex
	isRunning bool
	shutdown  chan struct{}
}

// TraceProvider wraps OpenTelemetry trace provider with enhanced functionality
type TraceProvider struct {
	provider trace.TracerProvider
	tracers  map[string]trace.Tracer
	mu       sync.RWMutex
}

// MetricProvider wraps OpenTelemetry metric provider with enhanced functionality
type MetricProvider struct {
	provider metric.MeterProvider
	meters   map[string]metric.Meter
	mu       sync.RWMutex
}

// LogProvider handles structured logging with OpenTelemetry integration
type LogProvider struct {
	provider interface{} // Will be *sdklog.LoggerProvider when available
	logger   logger.Logger
}

// NewManager creates a new observability manager
func NewManager(lc fx.Lifecycle, logger logger.Logger) (*Manager, error) {
	config := LoadObservabilityConfig()

	om := &Manager{
		config:   config,
		logger:   logger,
		shutdown: make(chan struct{}),
	}

	if !config.Enabled {
		logger.Info(context.Background(), "Observability disabled by configuration")
		return om, nil
	}

	// Initialize resource
	if err := om.initResource(); err != nil {
		return nil, fmt.Errorf("failed to initialize resource: %w", err)
	}

	// Initialize providers
	if err := om.initProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Initialize components
	if err := om.initComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Register lifecycle hooks
	lc.Append(fx.Hook{
		OnStart: om.Start,
		OnStop:  om.Stop,
	})

	return om, nil
}

// initResource initializes the OpenTelemetry resource
func (om *Manager) initResource() error {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(om.config.ServiceName),
		semconv.ServiceVersion(om.config.Version),
		attribute.String("environment", om.config.Environment),
		attribute.String("service.namespace", om.config.Resource.ServiceNamespace),
		attribute.String("service.instance.id", om.config.Resource.ServiceInstance),
	}

	// Add deployment environment
	if om.config.Resource.DeploymentEnvironment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", om.config.Resource.DeploymentEnvironment))
	}

	// Add Kubernetes attributes if available
	if om.config.Resource.K8sPodName != "" {
		attrs = append(attrs, semconv.K8SPodName(om.config.Resource.K8sPodName))
	}
	if om.config.Resource.K8sPodIP != "" {
		attrs = append(attrs, attribute.String("k8s.pod.ip", om.config.Resource.K8sPodIP))
	}
	if om.config.Resource.K8sNamespace != "" {
		attrs = append(attrs, semconv.K8SNamespaceName(om.config.Resource.K8sNamespace))
	}
	if om.config.Resource.K8sNodeName != "" {
		attrs = append(attrs, semconv.K8SNodeName(om.config.Resource.K8sNodeName))
	}
	if om.config.Resource.K8sClusterName != "" {
		attrs = append(attrs, semconv.K8SClusterName(om.config.Resource.K8sClusterName))
	}

	// Add container attributes if available
	if om.config.Resource.ContainerName != "" {
		attrs = append(attrs, semconv.ContainerName(om.config.Resource.ContainerName))
	}
	if om.config.Resource.ContainerID != "" {
		attrs = append(attrs, semconv.ContainerID(om.config.Resource.ContainerID))
	}

	// Add custom attributes
	for key, value := range om.config.Resource.CustomAttributes {
		attrs = append(attrs, attribute.String(key, value))
	}

	// Add runtime information
	attrs = append(attrs,
		semconv.ProcessRuntimeName("go"),
		semconv.ProcessRuntimeVersion(runtime.Version()),
		semconv.ProcessRuntimeDescription("Go runtime"),
	)

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(attrs...),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	om.resource = res
	return nil
}

// initProviders initializes all OpenTelemetry providers
func (om *Manager) initProviders() error {
	var err error

	// Initialize trace provider
	if om.config.Traces.Enabled {
		om.tracerProvider, err = NewTraceProvider(om.config, om.resource, om.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize trace provider: %w", err)
		}

		// Set global tracer provider
		otel.SetTracerProvider(om.tracerProvider.provider)

		// Set global propagator
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	// Initialize metric provider
	if om.config.Metrics.Enabled {
		om.meterProvider, err = NewMetricProvider(om.config, om.resource, om.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize metric provider: %w", err)
		}

		// Set global meter provider
		otel.SetMeterProvider(om.meterProvider.provider)
	}

	// Initialize log provider
	if om.config.Logs.Enabled {
		om.loggerProvider, err = NewLogProvider(om.config, om.resource, om.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize log provider: %w", err)
		}
	}

	return nil
}

// initComponents initializes observability components
func (om *Manager) initComponents() error {
	var err error

	// Initialize instrumentor
	om.instrumentor, err = NewInstrumentor(om, om.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize instrumentor: %w", err)
	}

	// Initialize metric collector
	if om.config.Metrics.Enabled {
		om.collector, err = NewMetricCollector(om, om.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize metric collector: %w", err)
		}
	}

	return nil
}

// Start starts the observability manager
func (om *Manager) Start(ctx context.Context) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	if !om.config.Enabled {
		return nil
	}

	om.isRunning = true

	// Start metric collection
	if om.collector != nil {
		if err := om.collector.Start(ctx); err != nil {
			return fmt.Errorf("failed to start metric collector: %w", err)
		}
	}

	om.logger.Info(ctx, "Observability manager started", map[string]interface{}{
		"service_name": om.config.ServiceName,
		"version":      om.config.Version,
		"environment":  om.config.Environment,
		"endpoint":     om.config.Endpoint,
		"traces":       om.config.Traces.Enabled,
		"metrics":      om.config.Metrics.Enabled,
		"logs":         om.config.Logs.Enabled,
	})

	return nil
}

// Stop stops the observability manager
func (om *Manager) Stop(ctx context.Context) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	if !om.isRunning {
		return nil
	}

	om.logger.Info(ctx, "Stopping observability manager...")

	// Stop metric collection
	if om.collector != nil {
		if err := om.collector.Stop(ctx); err != nil {
			om.logger.Error(ctx, "Failed to stop metric collector", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Shutdown providers
	if om.tracerProvider != nil {
		if err := om.tracerProvider.Shutdown(ctx); err != nil {
			om.logger.Error(ctx, "Failed to shutdown trace provider", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if om.meterProvider != nil {
		if err := om.meterProvider.Shutdown(ctx); err != nil {
			om.logger.Error(ctx, "Failed to shutdown metric provider", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if om.loggerProvider != nil {
		if err := om.loggerProvider.Shutdown(ctx); err != nil {
			om.logger.Error(ctx, "Failed to shutdown log provider", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	om.isRunning = false
	close(om.shutdown)

	om.logger.Info(ctx, "Observability manager stopped")
	return nil
}

// GetTracer returns a tracer for the given name
func (om *Manager) GetTracer(name string) trace.Tracer {
	if om.tracerProvider == nil {
		return noop.NewTracerProvider().Tracer(name)
	}
	return om.tracerProvider.GetTracer(name)
}

// GetMeter returns a meter for the given name
func (om *Manager) GetMeter(name string) metric.Meter {
	if om.meterProvider == nil {
		// Return a noop meter for disabled observability
		return nil
	}
	return om.meterProvider.GetMeter(name)
}

// NewTraceProvider creates a new trace provider with OTLP exporter
func NewTraceProvider(config *Config, res *resource.Resource, logger logger.Logger) (*TraceProvider, error) {
	ctx := context.Background()

	// Create OTLP trace exporter (HTTP)
	opts := []otlptrace.Option{
		otlptrace.WithEndpoint(config.Endpoint), // Use endpoint as-is (already includes scheme)
		otlptrace.WithTimeout(config.Timeout),
	}
	if config.Insecure {
		opts = append(opts, otlptrace.WithInsecure())
	}
	exporter, err := otlptrace.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(config.Traces.BatchTimeout),
			sdktrace.WithMaxExportBatchSize(config.Traces.MaxExportBatch),
			sdktrace.WithMaxQueueSize(config.Traces.QueueSize),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(createSampler(config.Traces.Sampling)),
	)

	return &TraceProvider{
		provider: provider,
		tracers:  make(map[string]trace.Tracer),
	}, nil
}

// NewMetricProvider creates a new metric provider with OTLP exporter
func NewMetricProvider(config *Config, res *resource.Resource, logger logger.Logger) (*MetricProvider, error) {
	ctx := context.Background()

	// Create OTLP metric exporter (HTTP)
	opts := []otlpmetric.Option{
		otlpmetric.WithEndpoint(config.Endpoint), // Use endpoint as-is (already includes scheme)
		otlpmetric.WithTimeout(config.Timeout),
	}
	if config.Insecure {
		opts = append(opts, otlpmetric.WithInsecure())
	}
	exporter, err := otlpmetric.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create metric provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(config.Metrics.DefaultInterval),
		)),
		sdkmetric.WithResource(res),
	)

	return &MetricProvider{
		provider: provider,
		meters:   make(map[string]metric.Meter),
	}, nil
}

// NewLogProvider creates a new log provider (simplified for now)
func NewLogProvider(config *Config, res *resource.Resource, logger logger.Logger) (*LogProvider, error) {
	// For now, just return a basic log provider
	// OTEL logs are still experimental, so we'll focus on trace correlation
	return &LogProvider{
		provider: nil, // Will be implemented when OTEL logs stabilize
		logger:   logger,
	}, nil
}

// createSampler creates a sampler based on configuration
func createSampler(sampling SamplingConfig) sdktrace.Sampler {
	switch sampling.Type {
	case "always":
		return sdktrace.AlwaysSample()
	case "never":
		return sdktrace.NeverSample()
	case "ratio":
		return sdktrace.TraceIDRatioBased(sampling.Rate)
	case "parent_based":
		// Use ParentBased with TraceIDRatioBased as root sampler
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampling.Rate))
	case "":
		// Default to ratio-based sampling if no type specified
		return sdktrace.TraceIDRatioBased(sampling.Rate)
	default:
		// For unknown types, default to ratio-based sampling
		return sdktrace.TraceIDRatioBased(sampling.Rate)
	}
}

// GetTracer returns a tracer for the given name
func (tp *TraceProvider) GetTracer(name string) trace.Tracer {
	tp.mu.RLock()
	if tracer, exists := tp.tracers[name]; exists {
		tp.mu.RUnlock()
		return tracer
	}
	tp.mu.RUnlock()

	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Double-check after acquiring write lock
	if tracer, exists := tp.tracers[name]; exists {
		return tracer
	}

	tracer := tp.provider.Tracer(name)
	tp.tracers[name] = tracer
	return tracer
}

// Shutdown gracefully shuts down the trace provider
func (tp *TraceProvider) Shutdown(ctx context.Context) error {
	if shutdownable, ok := tp.provider.(interface{ Shutdown(context.Context) error }); ok {
		return shutdownable.Shutdown(ctx)
	}
	return nil
}

// GetMeter returns a meter for the given name
func (mp *MetricProvider) GetMeter(name string) metric.Meter {
	mp.mu.RLock()
	if meter, exists := mp.meters[name]; exists {
		mp.mu.RUnlock()
		return meter
	}
	mp.mu.RUnlock()

	mp.mu.Lock()
	defer mp.mu.Unlock()

	// Double-check after acquiring write lock
	if meter, exists := mp.meters[name]; exists {
		return meter
	}

	meter := mp.provider.Meter(name)
	mp.meters[name] = meter
	return meter
}

// Shutdown gracefully shuts down the metric provider
func (mp *MetricProvider) Shutdown(ctx context.Context) error {
	if shutdownable, ok := mp.provider.(interface{ Shutdown(context.Context) error }); ok {
		return shutdownable.Shutdown(ctx)
	}
	return nil
}

// Shutdown gracefully shuts down the log provider
func (lp *LogProvider) Shutdown(ctx context.Context) error {
	if shutdownable, ok := lp.provider.(interface{ Shutdown(context.Context) error }); ok {
		return shutdownable.Shutdown(ctx)
	}
	return nil
}

// GetInstrumentor returns the instrumentor
func (om *Manager) GetInstrumentor() *Instrumentor {
	return om.instrumentor
}

// GetConfig returns the observability configuration
func (om *Manager) GetConfig() *Config {
	return om.config
}

// IsEnabled returns whether observability is enabled
func (om *Manager) IsEnabled() bool {
	return om.config.Enabled
}

// IsRunning returns whether the manager is running
func (om *Manager) IsRunning() bool {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.isRunning
}
