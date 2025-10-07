package observability

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
)

// Config holds comprehensive observability configuration
type Config struct {
	// General settings
	Enabled     bool   `json:"enabled"`
	ServiceName string `json:"service_name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`

	// Export settings
	Endpoint         string        `json:"endpoint"`
	ExporterProtocol string        `json:"exporter_protocol"` // grpc, http
	Insecure         bool          `json:"insecure"`
	Timeout          time.Duration `json:"timeout"`
	Compression      string        `json:"compression"` // gzip, none

	// Resource attributes
	Resource ResourceConfig `json:"resource"`

	// Component-specific settings
	Traces  TracesConfig  `json:"traces"`
	Metrics MetricsConfig `json:"metrics"`
	Logs    LogsConfig    `json:"logs"`

	// Performance settings
	Performance PerformanceConfig `json:"performance"`

	// Features
	Features FeaturesConfig `json:"features"`
}

// ResourceConfig defines resource attributes
type ResourceConfig struct {
	// Service attributes
	ServiceNamespace string `json:"service_namespace"`
	ServiceInstance  string `json:"service_instance"`

	// Deployment attributes
	DeploymentEnvironment string `json:"deployment_environment"`

	// K8s attributes (auto-detected if available)
	K8sPodName     string `json:"k8s_pod_name"`
	K8sPodIP       string `json:"k8s_pod_ip"`
	K8sNamespace   string `json:"k8s_namespace"`
	K8sNodeName    string `json:"k8s_node_name"`
	K8sClusterName string `json:"k8s_cluster_name"`

	// Container attributes
	ContainerName string `json:"container_name"`
	ContainerID   string `json:"container_id"`

	// Custom attributes
	CustomAttributes map[string]string `json:"custom_attributes"`
}

// TracesConfig configures tracing behavior
type TracesConfig struct {
	Enabled bool `json:"enabled"`

	// Sampling configuration
	Sampling SamplingConfig `json:"sampling"`

	// Span limits
	MaxAttributesPerSpan int `json:"max_attributes_per_span"`
	MaxEventsPerSpan     int `json:"max_events_per_span"`
	MaxLinksPerSpan      int `json:"max_links_per_span"`

	// Span processors
	BatchTimeout   time.Duration `json:"batch_timeout"`
	BatchSize      int           `json:"batch_size"`
	QueueSize      int           `json:"queue_size"`
	MaxExportBatch int           `json:"max_export_batch"`

	// Filtering
	ExcludedPaths []string `json:"excluded_paths"`
}

// SamplingConfig defines sampling strategies
type SamplingConfig struct {
	Type string  `json:"type"` // always, never, ratio, parent_based
	Rate float64 `json:"rate"` // 0.0 to 1.0 for ratio sampling
}

// MetricsConfig configures metrics behavior
type MetricsConfig struct {
	Enabled bool `json:"enabled"`

	// Collection intervals
	DefaultInterval time.Duration `json:"default_interval"`
	RuntimeInterval time.Duration `json:"runtime_interval"`

	// Metric types to collect
	HTTP     bool `json:"http"`
	Database bool `json:"database"`
	Redis    bool `json:"redis"`
	AMQP     bool `json:"amqp"`
	Runtime  bool `json:"runtime"`
	Business bool `json:"business"`

	// Resource metrics
	CPU    bool `json:"cpu"`
	Memory bool `json:"memory"`
	Disk   bool `json:"disk"`

	// Histogram boundaries
	HTTPLatencyBoundaries []float64 `json:"http_latency_boundaries"`
	DBLatencyBoundaries   []float64 `json:"db_latency_boundaries"`
}

// LogsConfig configures logging behavior
type LogsConfig struct {
	Enabled bool `json:"enabled"`

	// Correlation
	TraceCorrelation bool `json:"trace_correlation"`
	SpanCorrelation  bool `json:"span_correlation"`

	// Log levels to export
	ExportLevels []string `json:"export_levels"` // debug, info, warn, error

	// Batch processing
	BatchTimeout time.Duration `json:"batch_timeout"`
	BatchSize    int           `json:"batch_size"`
	QueueSize    int           `json:"queue_size"`

	// Structured logging
	StructuredFields bool              `json:"structured_fields"`
	CustomFields     map[string]string `json:"custom_fields"`
}

// PerformanceConfig optimizes performance
type PerformanceConfig struct {
	// Memory optimization
	MaxMemoryUsage     int64 `json:"max_memory_usage"`     // bytes
	MemoryLimitPercent int   `json:"memory_limit_percent"` // 0-100

	// CPU optimization
	MaxCPUUsage     float64 `json:"max_cpu_usage"` // 0.0-1.0
	WorkerPoolSize  int     `json:"worker_pool_size"`
	QueueBufferSize int     `json:"queue_buffer_size"`

	// Network optimization
	MaxBatchSize   int           `json:"max_batch_size"`
	FlushTimeout   time.Duration `json:"flush_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryBackoff   time.Duration `json:"retry_backoff"`
	ConnectionPool int           `json:"connection_pool"`

	// Adaptive behavior
	AdaptiveSampling   bool    `json:"adaptive_sampling"`
	ErrorSamplingBoost float64 `json:"error_sampling_boost"` // multiply sampling rate on errors
}

// FeaturesConfig enables/disables specific features
type FeaturesConfig struct {
	// Automatic instrumentation
	AutoHTTP     bool `json:"auto_http"`
	AutoDatabase bool `json:"auto_database"`
	AutoRedis    bool `json:"auto_redis"`
	AutoAMQP     bool `json:"auto_amqp"`

	// Advanced features
	DistributedTracing bool `json:"distributed_tracing"`
	ErrorTracking      bool `json:"error_tracking"`
	SecurityEvents     bool `json:"security_events"`
	PerformanceMonitor bool `json:"performance_monitor"`
	BusinessMetrics    bool `json:"business_metrics"`

	// Health and diagnostics
	HealthChecks    bool `json:"health_checks"`
	ReadinessProbes bool `json:"readiness_probes"`
	LivenessProbes  bool `json:"liveness_probes"`

	// Development features
	DebugMode bool `json:"debug_mode"`
	DryRun    bool `json:"dry_run"` // simulate without sending
}

// LoadObservabilityConfig loads configuration from environment variables
func LoadObservabilityConfig() *Config {
	return &Config{
		// General settings
		Enabled:     getBoolEnv("SIGNOZ_ENABLED", "OTEL_TRACES_ENABLED", false),
		ServiceName: getStringEnv("OTEL_SERVICE_NAME", "spooliq-api"),
		Version:     getStringEnv("VERSION", "OTEL_SERVICE_VERSION", "1.0.0"),
		Environment: getStringEnv("ENV", "DEPLOYMENT_ENVIRONMENT", config.EnvironmentConfig()),

		// Export settings
		Endpoint:         stripURLScheme(getStringEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")),
		ExporterProtocol: getStringEnv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc"), // Default to gRPC
		Insecure:         getBoolEnv("OTEL_EXPORTER_OTLP_INSECURE", true),
		Timeout:          getDurationEnv("OTEL_EXPORTER_OTLP_TIMEOUT", 10*time.Second),
		Compression:      getStringEnv("OTEL_EXPORTER_OTLP_COMPRESSION", "gzip"),

		// Resource configuration
		Resource: loadResourceConfig(),

		// Component configurations
		Traces:  loadTracesConfig(),
		Metrics: loadMetricsConfig(),
		Logs:    loadLogsConfig(),

		// Performance configuration
		Performance: loadPerformanceConfig(),

		// Features configuration
		Features: loadFeaturesConfig(),
	}
}

// loadResourceConfig loads resource configuration
func loadResourceConfig() ResourceConfig {
	return ResourceConfig{
		ServiceNamespace:      getStringEnv("OTEL_SERVICE_NAMESPACE", "spooliq"),
		ServiceInstance:       getStringEnv("OTEL_SERVICE_INSTANCE", getHostname()),
		DeploymentEnvironment: getStringEnv("DEPLOYMENT_ENVIRONMENT", config.EnvironmentConfig()),

		// K8s attributes (auto-detected)
		K8sPodName:     getStringEnv("POD_NAME", "K8S_POD_NAME"),
		K8sPodIP:       getStringEnv("POD_IP", "K8S_POD_IP"),
		K8sNamespace:   getStringEnv("POD_NAMESPACE", "K8S_NAMESPACE"),
		K8sNodeName:    getStringEnv("NODE_NAME", "K8S_NODE_NAME"),
		K8sClusterName: getStringEnv("K8S_CLUSTER_NAME"),

		// Container attributes
		ContainerName: getStringEnv("CONTAINER_NAME"),
		ContainerID:   getStringEnv("CONTAINER_ID"),

		// Custom attributes from OTEL_RESOURCE_ATTRIBUTES
		CustomAttributes: parseResourceAttributes(),
	}
}

// loadTracesConfig loads traces configuration
func loadTracesConfig() TracesConfig {
	return TracesConfig{
		Enabled: getBoolEnv("OTEL_TRACES_ENABLED", true),

		Sampling: SamplingConfig{
			Type: getStringEnv("OTEL_TRACES_SAMPLER", "ratio"),
			Rate: getFloat64Env("OTEL_TRACES_SAMPLER_ARG", getDefaultSamplingRate()),
		},

		MaxAttributesPerSpan: getIntEnv("OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT", 128),
		MaxEventsPerSpan:     getIntEnv("OTEL_SPAN_EVENT_COUNT_LIMIT", 128),
		MaxLinksPerSpan:      getIntEnv("OTEL_SPAN_LINK_COUNT_LIMIT", 128),

		BatchTimeout:   getDurationEnv("OTEL_BSP_SCHEDULE_DELAY", 5*time.Second),
		BatchSize:      getIntEnv("OTEL_BSP_MAX_EXPORT_BATCH_SIZE", 512),
		QueueSize:      getIntEnv("OTEL_BSP_MAX_QUEUE_SIZE", 2048),
		MaxExportBatch: getIntEnv("OTEL_BSP_EXPORT_BATCH_SIZE", 512),

		ExcludedPaths: getStringSliceEnv("OTEL_TRACES_EXCLUDED_PATHS", []string{
			"/health", "/health_check", "/metrics", "/ready", "/live",
		}),
	}
}

// loadMetricsConfig loads metrics configuration
func loadMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled: getBoolEnv("OTEL_METRICS_ENABLED", true),

		DefaultInterval: getDurationEnv("OTEL_METRIC_EXPORT_INTERVAL", 30*time.Second),
		RuntimeInterval: getDurationEnv("OTEL_RUNTIME_METRIC_INTERVAL", 10*time.Second),

		HTTP:     getBoolEnv("OTEL_METRICS_HTTP_ENABLED", true),
		Database: getBoolEnv("OTEL_METRICS_DATABASE_ENABLED", true),
		Redis:    getBoolEnv("OTEL_METRICS_REDIS_ENABLED", true),
		AMQP:     getBoolEnv("OTEL_METRICS_AMQP_ENABLED", true),
		Runtime:  getBoolEnv("OTEL_METRICS_RUNTIME_ENABLED", true),
		Business: getBoolEnv("OTEL_METRICS_BUSINESS_ENABLED", true),

		CPU:    getBoolEnv("OTEL_METRICS_CPU_ENABLED", true),
		Memory: getBoolEnv("OTEL_METRICS_MEMORY_ENABLED", true),
		Disk:   getBoolEnv("OTEL_METRICS_DISK_ENABLED", false),

		HTTPLatencyBoundaries: getFloat64SliceEnv("OTEL_HTTP_LATENCY_BOUNDARIES",
			[]float64{0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10.0}),
		DBLatencyBoundaries: getFloat64SliceEnv("OTEL_DB_LATENCY_BOUNDARIES",
			[]float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0}),
	}
}

// loadLogsConfig loads logs configuration
func loadLogsConfig() LogsConfig {
	return LogsConfig{
		Enabled: getBoolEnv("OTEL_LOGS_ENABLED", true),

		TraceCorrelation: getBoolEnv("OTEL_LOGS_TRACE_CORRELATION", true),
		SpanCorrelation:  getBoolEnv("OTEL_LOGS_SPAN_CORRELATION", true),

		ExportLevels: getStringSliceEnv("OTEL_LOGS_EXPORT_LEVELS", []string{"info", "warn", "error"}),

		BatchTimeout: getDurationEnv("OTEL_BLRP_SCHEDULE_DELAY", 5*time.Second),
		BatchSize:    getIntEnv("OTEL_BLRP_MAX_EXPORT_BATCH_SIZE", 512),
		QueueSize:    getIntEnv("OTEL_BLRP_MAX_QUEUE_SIZE", 2048),

		StructuredFields: getBoolEnv("OTEL_LOGS_STRUCTURED", true),
		CustomFields:     parseCustomFields(),
	}
}

// loadPerformanceConfig loads performance configuration
func loadPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		MaxMemoryUsage:     getInt64Env("OTEL_MAX_MEMORY_USAGE", 128*1024*1024), // 128MB
		MemoryLimitPercent: getIntEnv("OTEL_MEMORY_LIMIT_PERCENT", 10),          // 10%

		MaxCPUUsage:     getFloat64Env("OTEL_MAX_CPU_USAGE", 0.1), // 10%
		WorkerPoolSize:  getIntEnv("OTEL_WORKER_POOL_SIZE", 4),
		QueueBufferSize: getIntEnv("OTEL_QUEUE_BUFFER_SIZE", 1000),

		MaxBatchSize:   getIntEnv("OTEL_MAX_BATCH_SIZE", 1000),
		FlushTimeout:   getDurationEnv("OTEL_FLUSH_TIMEOUT", 5*time.Second),
		RetryAttempts:  getIntEnv("OTEL_RETRY_ATTEMPTS", 3),
		RetryBackoff:   getDurationEnv("OTEL_RETRY_BACKOFF", 1*time.Second),
		ConnectionPool: getIntEnv("OTEL_CONNECTION_POOL", 5),

		AdaptiveSampling:   getBoolEnv("OTEL_ADAPTIVE_SAMPLING", true),
		ErrorSamplingBoost: getFloat64Env("OTEL_ERROR_SAMPLING_BOOST", 5.0),
	}
}

// loadFeaturesConfig loads features configuration
func loadFeaturesConfig() FeaturesConfig {
	return FeaturesConfig{
		AutoHTTP:     getBoolEnv("OTEL_AUTO_HTTP", true),
		AutoDatabase: getBoolEnv("OTEL_AUTO_DATABASE", true),
		AutoRedis:    getBoolEnv("OTEL_AUTO_REDIS", true),
		AutoAMQP:     getBoolEnv("OTEL_AUTO_AMQP", true),

		DistributedTracing: getBoolEnv("OTEL_DISTRIBUTED_TRACING", true),
		ErrorTracking:      getBoolEnv("OTEL_ERROR_TRACKING", true),
		SecurityEvents:     getBoolEnv("OTEL_SECURITY_EVENTS", false),
		PerformanceMonitor: getBoolEnv("OTEL_PERFORMANCE_MONITOR", true),
		BusinessMetrics:    getBoolEnv("OTEL_BUSINESS_METRICS", true),

		HealthChecks:    getBoolEnv("OTEL_HEALTH_CHECKS", true),
		ReadinessProbes: getBoolEnv("OTEL_READINESS_PROBES", true),
		LivenessProbes:  getBoolEnv("OTEL_LIVENESS_PROBES", true),

		DebugMode: getBoolEnv("OTEL_DEBUG_MODE", config.EnvironmentConfig() == "development"),
		DryRun:    getBoolEnv("OTEL_DRY_RUN", false),
	}
}

// Helper functions for environment variable parsing

func getStringEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	if len(keys) > 0 {
		return ""
	}
	return keys[len(keys)-1] // default value is the last argument
}

func getBoolEnv(keys ...interface{}) bool {
	var defaultValue bool
	var envKeys []string

	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := key.(bool); ok {
				defaultValue = v
				break
			}
		}
		if k, ok := key.(string); ok {
			envKeys = append(envKeys, k)
		}
	}

	for _, key := range envKeys {
		if value := os.Getenv(key); value != "" {
			return value == "true" || value == "1" || value == "yes"
		}
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Try parsing as milliseconds
		if ms, err := strconv.Atoi(value); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultValue
}

func getStringSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getFloat64SliceEnv(key string, defaultValue []float64) []float64 {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]float64, 0, len(parts))
		for _, part := range parts {
			if f, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
				result = append(result, f)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

func getDefaultSamplingRate() float64 {
	env := config.EnvironmentConfig()
	switch env {
	case "production":
		return 0.1 // 10% sampling in production
	case "staging":
		return 0.5 // 50% sampling in staging
	default:
		return 1.0 // 100% sampling in development
	}
}

func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

func parseResourceAttributes() map[string]string {
	attributes := make(map[string]string)
	if value := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); value != "" {
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				attributes[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return attributes
}

func parseCustomFields() map[string]string {
	fields := make(map[string]string)
	if value := os.Getenv("OTEL_LOGS_CUSTOM_FIELDS"); value != "" {
		pairs := strings.Split(value, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				fields[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return fields
}

// stripURLScheme removes the scheme (http://, https://) from an endpoint URL
// This is needed because OTLP HTTP exporters expect "host:port" format without scheme
func stripURLScheme(endpoint string) string {
	if endpoint == "" {
		return endpoint
	}

	// Parse the URL to extract host and port
	if parsedURL, err := url.Parse(endpoint); err == nil && parsedURL.Host != "" {
		return parsedURL.Host
	}

	// If parsing fails or no host found, return original endpoint
	// This handles cases where endpoint is already in "host:port" format
	return endpoint
}
