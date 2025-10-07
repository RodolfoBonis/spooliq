package observability

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricCollector collects and manages custom metrics
type MetricCollector struct {
	manager *Manager
	logger  logger.Logger
	config  *Config

	// Runtime metrics
	runtimeCollector *RuntimeCollector

	// Business metrics
	businessCollector *BusinessCollector

	// Performance metrics
	performanceCollector *PerformanceCollector

	// System metrics
	systemCollector *SystemCollector

	// Control
	mu       sync.RWMutex
	running  bool
	stopChan chan struct{}
}

// RuntimeCollector collects Go runtime metrics
type RuntimeCollector struct {
	meter metric.Meter

	// Memory metrics
	memAlloc     metric.Int64Gauge
	memSys       metric.Int64Gauge
	memHeapAlloc metric.Int64Gauge
	memHeapSys   metric.Int64Gauge
	memStack     metric.Int64Gauge
	memGCCount   metric.Int64Counter
	memGCPause   metric.Float64Histogram

	// Goroutine metrics
	goroutines metric.Int64Gauge

	// GC metrics
	gcCPUFraction metric.Float64Gauge
}

// BusinessCollector collects application-specific business metrics
type BusinessCollector struct {
	meter metric.Meter

	// Request metrics
	activeUsers  metric.Int64Gauge
	requestRate  metric.Float64Gauge
	errorRate    metric.Float64Gauge
	responseTime metric.Float64Histogram

	// Feature metrics
	featureUsage   metric.Int64Counter
	conversionRate metric.Float64Gauge
	retentionRate  metric.Float64Gauge

	// Custom business metrics
	customCounters   map[string]metric.Int64Counter
	customGauges     map[string]metric.Int64Gauge
	customHistograms map[string]metric.Float64Histogram
	mu               sync.RWMutex
}

// PerformanceCollector collects performance metrics
type PerformanceCollector struct {
	meter metric.Meter

	// Latency metrics
	p50Latency metric.Float64Gauge
	p90Latency metric.Float64Gauge
	p95Latency metric.Float64Gauge
	p99Latency metric.Float64Gauge

	// Throughput metrics
	requestsPerSecond metric.Float64Gauge
	messagesPerSecond metric.Float64Gauge

	// Resource utilization
	cpuUtilization    metric.Float64Gauge
	memoryUtilization metric.Float64Gauge
	diskUtilization   metric.Float64Gauge

	// Cache metrics
	cacheHitRate  metric.Float64Gauge
	cacheMissRate metric.Float64Gauge
}

// SystemCollector collects system-level metrics
type SystemCollector struct {
	meter metric.Meter

	// Connection pools
	dbConnections    metric.Int64Gauge
	redisConnections metric.Int64Gauge
	httpConnections  metric.Int64Gauge

	// Queue metrics
	queueDepth metric.Int64Gauge
	queueRate  metric.Float64Gauge

	// Health metrics
	healthScore metric.Float64Gauge
	uptime      metric.Int64Gauge
}

// NewMetricCollector creates a new metric collector
func NewMetricCollector(manager *Manager, logger logger.Logger) (*MetricCollector, error) {
	mc := &MetricCollector{
		manager:  manager,
		logger:   logger,
		config:   manager.GetConfig(),
		stopChan: make(chan struct{}),
	}

	if !manager.IsEnabled() {
		return mc, nil
	}

	var err error

	// Initialize runtime collector
	if mc.config.Metrics.Runtime {
		mc.runtimeCollector, err = NewRuntimeCollector(manager)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize runtime collector: %w", err)
		}
	}

	// Initialize business collector
	if mc.config.Metrics.Business {
		mc.businessCollector, err = NewBusinessCollector(manager)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize business collector: %w", err)
		}
	}

	// Initialize performance collector
	mc.performanceCollector, err = NewPerformanceCollector(manager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize performance collector: %w", err)
	}

	// Initialize system collector
	mc.systemCollector, err = NewSystemCollector(manager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize system collector: %w", err)
	}

	return mc, nil
}

// Start starts the metric collection
func (mc *MetricCollector) Start(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return fmt.Errorf("metric collector is already running")
	}

	mc.running = true

	// Start runtime metrics collection
	if mc.runtimeCollector != nil {
		go mc.collectRuntimeMetrics(ctx)
	}

	// Start business metrics collection
	if mc.businessCollector != nil {
		go mc.collectBusinessMetrics(ctx)
	}

	// Start performance metrics collection
	if mc.performanceCollector != nil {
		go mc.collectPerformanceMetrics(ctx)
	}

	// Start system metrics collection
	if mc.systemCollector != nil {
		go mc.collectSystemMetrics(ctx)
	}

	mc.logger.Info(ctx, "Metric collector started")
	return nil
}

// Stop stops the metric collection
func (mc *MetricCollector) Stop(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.running {
		return nil
	}

	close(mc.stopChan)
	mc.running = false

	mc.logger.Info(ctx, "Metric collector stopped")
	return nil
}

// NewRuntimeCollector creates a new runtime metrics collector
func NewRuntimeCollector(manager *Manager) (*RuntimeCollector, error) {
	meter := manager.GetMeter("runtime")

	rc := &RuntimeCollector{meter: meter}

	var err error

	// Memory metrics
	rc.memAlloc, err = meter.Int64Gauge(
		"go_memory_alloc_bytes",
		metric.WithDescription("Current allocated memory in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	rc.memSys, err = meter.Int64Gauge(
		"go_memory_sys_bytes",
		metric.WithDescription("Total system memory in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	rc.memHeapAlloc, err = meter.Int64Gauge(
		"go_memory_heap_alloc_bytes",
		metric.WithDescription("Current heap allocated memory in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	rc.memHeapSys, err = meter.Int64Gauge(
		"go_memory_heap_sys_bytes",
		metric.WithDescription("Total heap system memory in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	rc.memStack, err = meter.Int64Gauge(
		"go_memory_stack_bytes",
		metric.WithDescription("Current stack memory in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	rc.memGCCount, err = meter.Int64Counter(
		"go_gc_collections_total",
		metric.WithDescription("Total number of GC collections"),
	)
	if err != nil {
		return nil, err
	}

	rc.memGCPause, err = meter.Float64Histogram(
		"go_gc_pause_seconds",
		metric.WithDescription("GC pause duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Goroutine metrics
	rc.goroutines, err = meter.Int64Gauge(
		"go_goroutines",
		metric.WithDescription("Current number of goroutines"),
	)
	if err != nil {
		return nil, err
	}

	// GC metrics
	rc.gcCPUFraction, err = meter.Float64Gauge(
		"go_gc_cpu_fraction",
		metric.WithDescription("Fraction of CPU time used by GC"),
	)
	if err != nil {
		return nil, err
	}

	return rc, nil
}

// NewBusinessCollector creates a new business metrics collector
func NewBusinessCollector(manager *Manager) (*BusinessCollector, error) {
	meter := manager.GetMeter("business")

	bc := &BusinessCollector{
		meter:            meter,
		customCounters:   make(map[string]metric.Int64Counter),
		customGauges:     make(map[string]metric.Int64Gauge),
		customHistograms: make(map[string]metric.Float64Histogram),
	}

	var err error

	// Request metrics
	bc.activeUsers, err = meter.Int64Gauge(
		"active_users",
		metric.WithDescription("Current number of active users"),
	)
	if err != nil {
		return nil, err
	}

	bc.requestRate, err = meter.Float64Gauge(
		"request_rate",
		metric.WithDescription("Current request rate per second"),
		metric.WithUnit("1/s"),
	)
	if err != nil {
		return nil, err
	}

	bc.errorRate, err = meter.Float64Gauge(
		"error_rate",
		metric.WithDescription("Current error rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	bc.responseTime, err = meter.Float64Histogram(
		"response_time_seconds",
		metric.WithDescription("Response time distribution"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Feature metrics
	bc.featureUsage, err = meter.Int64Counter(
		"feature_usage_total",
		metric.WithDescription("Total feature usage count"),
	)
	if err != nil {
		return nil, err
	}

	bc.conversionRate, err = meter.Float64Gauge(
		"conversion_rate",
		metric.WithDescription("Current conversion rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	bc.retentionRate, err = meter.Float64Gauge(
		"retention_rate",
		metric.WithDescription("Current retention rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// NewPerformanceCollector creates a new performance metrics collector
func NewPerformanceCollector(manager *Manager) (*PerformanceCollector, error) {
	meter := manager.GetMeter("performance")

	pc := &PerformanceCollector{meter: meter}

	var err error

	// Latency metrics
	pc.p50Latency, err = meter.Float64Gauge(
		"latency_p50_seconds",
		metric.WithDescription("50th percentile latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	pc.p90Latency, err = meter.Float64Gauge(
		"latency_p90_seconds",
		metric.WithDescription("90th percentile latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	pc.p95Latency, err = meter.Float64Gauge(
		"latency_p95_seconds",
		metric.WithDescription("95th percentile latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	pc.p99Latency, err = meter.Float64Gauge(
		"latency_p99_seconds",
		metric.WithDescription("99th percentile latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Throughput metrics
	pc.requestsPerSecond, err = meter.Float64Gauge(
		"requests_per_second",
		metric.WithDescription("Current requests per second"),
		metric.WithUnit("1/s"),
	)
	if err != nil {
		return nil, err
	}

	pc.messagesPerSecond, err = meter.Float64Gauge(
		"messages_per_second",
		metric.WithDescription("Current messages per second"),
		metric.WithUnit("1/s"),
	)
	if err != nil {
		return nil, err
	}

	// Resource utilization
	pc.cpuUtilization, err = meter.Float64Gauge(
		"cpu_utilization_percent",
		metric.WithDescription("Current CPU utilization percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pc.memoryUtilization, err = meter.Float64Gauge(
		"memory_utilization_percent",
		metric.WithDescription("Current memory utilization percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pc.diskUtilization, err = meter.Float64Gauge(
		"disk_utilization_percent",
		metric.WithDescription("Current disk utilization percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	// Cache metrics
	pc.cacheHitRate, err = meter.Float64Gauge(
		"cache_hit_rate_percent",
		metric.WithDescription("Current cache hit rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	pc.cacheMissRate, err = meter.Float64Gauge(
		"cache_miss_rate_percent",
		metric.WithDescription("Current cache miss rate percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, err
	}

	return pc, nil
}

// NewSystemCollector creates a new system metrics collector
func NewSystemCollector(manager *Manager) (*SystemCollector, error) {
	meter := manager.GetMeter("system")

	sc := &SystemCollector{meter: meter}

	var err error

	// Connection pools
	sc.dbConnections, err = meter.Int64Gauge(
		"database_connections_active",
		metric.WithDescription("Current active database connections"),
	)
	if err != nil {
		return nil, err
	}

	sc.redisConnections, err = meter.Int64Gauge(
		"redis_connections_active",
		metric.WithDescription("Current active Redis connections"),
	)
	if err != nil {
		return nil, err
	}

	sc.httpConnections, err = meter.Int64Gauge(
		"http_connections_active",
		metric.WithDescription("Current active HTTP connections"),
	)
	if err != nil {
		return nil, err
	}

	// Queue metrics
	sc.queueDepth, err = meter.Int64Gauge(
		"queue_depth",
		metric.WithDescription("Current queue depth"),
	)
	if err != nil {
		return nil, err
	}

	sc.queueRate, err = meter.Float64Gauge(
		"queue_processing_rate",
		metric.WithDescription("Current queue processing rate"),
		metric.WithUnit("1/s"),
	)
	if err != nil {
		return nil, err
	}

	// Health metrics
	sc.healthScore, err = meter.Float64Gauge(
		"health_score",
		metric.WithDescription("Current health score (0-1)"),
	)
	if err != nil {
		return nil, err
	}

	sc.uptime, err = meter.Int64Gauge(
		"uptime_seconds",
		metric.WithDescription("Application uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return sc, nil
}

// Collection methods

func (mc *MetricCollector) collectRuntimeMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.config.Metrics.RuntimeInterval)
	defer ticker.Stop()

	var lastNumGC uint32
	var lastPauseTotal time.Duration

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Record memory metrics
			mc.runtimeCollector.memAlloc.Record(ctx, int64(m.Alloc))
			mc.runtimeCollector.memSys.Record(ctx, int64(m.Sys))
			mc.runtimeCollector.memHeapAlloc.Record(ctx, int64(m.HeapAlloc))
			mc.runtimeCollector.memHeapSys.Record(ctx, int64(m.HeapSys))
			mc.runtimeCollector.memStack.Record(ctx, int64(m.StackSys))

			// Record goroutines
			mc.runtimeCollector.goroutines.Record(ctx, int64(runtime.NumGoroutine()))

			// Record GC metrics
			mc.runtimeCollector.gcCPUFraction.Record(ctx, m.GCCPUFraction)

			// Record GC collections
			if m.NumGC > lastNumGC {
				mc.runtimeCollector.memGCCount.Add(ctx, int64(m.NumGC-lastNumGC))
				lastNumGC = m.NumGC
			}

			// Record GC pause time
			totalPauseNs := time.Duration(m.PauseTotalNs)
			if totalPauseNs > lastPauseTotal {
				pauseDiff := totalPauseNs - lastPauseTotal
				mc.runtimeCollector.memGCPause.Record(ctx, pauseDiff.Seconds())
				lastPauseTotal = totalPauseNs
			}
		}
	}
}

func (mc *MetricCollector) collectBusinessMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.config.Metrics.DefaultInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-ticker.C:
			// Collect business metrics here
			// This would integrate with your business logic
		}
	}
}

func (mc *MetricCollector) collectPerformanceMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.config.Metrics.DefaultInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-ticker.C:
			// Collect performance metrics here
			// This would integrate with your performance monitoring
		}
	}
}

func (mc *MetricCollector) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.config.Metrics.DefaultInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mc.stopChan:
			return
		case <-ticker.C:
			// Collect system metrics here
			// This would integrate with your system monitoring
		}
	}
}

// Business metric helpers

// RecordFeatureUsage records usage of a specific feature
func (bc *BusinessCollector) RecordFeatureUsage(ctx context.Context, feature string, userID string) {
	if bc.featureUsage != nil {
		attrs := []attribute.KeyValue{
			attribute.String("feature", feature),
			attribute.String("user_id", userID),
		}
		bc.featureUsage.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// CreateCustomCounter creates a custom business counter
func (bc *BusinessCollector) CreateCustomCounter(name, description string) (metric.Int64Counter, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if counter, exists := bc.customCounters[name]; exists {
		return counter, nil
	}

	counter, err := bc.meter.Int64Counter(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	bc.customCounters[name] = counter
	return counter, nil
}

// CreateCustomGauge creates a custom business gauge
func (bc *BusinessCollector) CreateCustomGauge(name, description string) (metric.Int64Gauge, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if gauge, exists := bc.customGauges[name]; exists {
		return gauge, nil
	}

	gauge, err := bc.meter.Int64Gauge(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	bc.customGauges[name] = gauge
	return gauge, nil
}

// CreateCustomHistogram creates a custom business histogram
func (bc *BusinessCollector) CreateCustomHistogram(name, description string) (metric.Float64Histogram, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if histogram, exists := bc.customHistograms[name]; exists {
		return histogram, nil
	}

	histogram, err := bc.meter.Float64Histogram(
		name,
		metric.WithDescription(description),
	)
	if err != nil {
		return nil, err
	}

	bc.customHistograms[name] = histogram
	return histogram, nil
}

// GetBusinessCollector returns the business collector
func (mc *MetricCollector) GetBusinessCollector() *BusinessCollector {
	return mc.businessCollector
}

// GetPerformanceCollector returns the performance collector
func (mc *MetricCollector) GetPerformanceCollector() *PerformanceCollector {
	return mc.performanceCollector
}

// GetSystemCollector returns the system collector
func (mc *MetricCollector) GetSystemCollector() *SystemCollector {
	return mc.systemCollector
}
