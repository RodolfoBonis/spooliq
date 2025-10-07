package observability

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"go.uber.org/fx"
)

// Module provides the observability module for dependency injection
var Module = fx.Module("observability",
	// Core providers
	fx.Provide(
		NewObservabilityManager,
		NewHelper,
		// Removed: NewObservabilityLogger, NewDecorator, NewAMQPInstrumentor, NewPerformanceOptimizer
	),

	// Lifecycle hooks
	fx.Invoke(func(lc fx.Lifecycle, manager *ObservabilityManager, helper *Helper) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// Set global helper for convenience functions
				SetGlobalHelper(helper)
				return nil
			},
		})
	}),

	// Export types for other modules
	fx.Provide(
		fx.Annotate(
			func(manager *ObservabilityManager) (*ObservabilityManager, error) {
				return manager, nil
			},
			fx.As(new(ObservabilityManagerInterface)),
		),
	),
)

// ObservabilityManagerInterface defines the interface for observability manager
type ObservabilityManagerInterface interface {
	IsEnabled() bool
	GetConfig() *ObservabilityConfig
	GetInstrumentor() *Instrumentor
}

// ProvideObservabilityComponents provides individual components
func ProvideObservabilityComponents() fx.Option {
	return fx.Options(
		Module,
		fx.Provide(
			// Individual component providers
			func(manager *ObservabilityManager) *Instrumentor {
				return manager.GetInstrumentor()
			},
			func(manager *ObservabilityManager, logger logger.Logger) (*MetricCollector, error) {
				return NewMetricCollector(manager, logger)
			},
		),
	)
}

// ProvideWithConfiguration provides observability with custom configuration
func ProvideWithConfiguration(configLoader func() *ObservabilityConfig) fx.Option {
	return fx.Options(
		fx.Provide(configLoader),
		fx.Provide(
			func(config *ObservabilityConfig, lc fx.Lifecycle, logger logger.Logger) (*ObservabilityManager, error) {
				// Create manager with custom config
				manager := &ObservabilityManager{
					config: config,
					logger: logger,
				}

				// Initialize based on config
				if config.Enabled {
					if err := manager.initResource(); err != nil {
						return nil, err
					}
					if err := manager.initProviders(); err != nil {
						return nil, err
					}
					if err := manager.initComponents(); err != nil {
						return nil, err
					}
				}

				// Register lifecycle
				lc.Append(fx.Hook{
					OnStart: manager.Start,
					OnStop:  manager.Stop,
				})

				return manager, nil
			},
		),
		fx.Provide(NewHelper),
		// Removed: fx.Provide(NewObservabilityLogger), fx.Provide(NewDecorator),
	)
}

// ProvideForTesting provides observability components for testing
func ProvideForTesting() fx.Option {
	return fx.Options(
		fx.Provide(
			func() *ObservabilityConfig {
				config := LoadObservabilityConfig()
				config.Enabled = false // Disable for testing
				config.Features.DryRun = true
				return config
			},
		),
		ProvideWithConfiguration(func() *ObservabilityConfig {
			config := LoadObservabilityConfig()
			config.Enabled = false
			return config
		}),
	)
}

// TracingOnlyModule provides only tracing components
var TracingOnlyModule = fx.Module("observability-tracing",
	fx.Provide(
		func() *ObservabilityConfig {
			config := LoadObservabilityConfig()
			config.Metrics.Enabled = false
			config.Logs.Enabled = false
			return config
		},
	),
	ProvideWithConfiguration(func() *ObservabilityConfig {
		config := LoadObservabilityConfig()
		config.Metrics.Enabled = false
		config.Logs.Enabled = false
		return config
	}),
)

// MetricsOnlyModule provides only metrics components
var MetricsOnlyModule = fx.Module("observability-metrics",
	fx.Provide(
		func() *ObservabilityConfig {
			config := LoadObservabilityConfig()
			config.Traces.Enabled = false
			config.Logs.Enabled = false
			return config
		},
	),
	ProvideWithConfiguration(func() *ObservabilityConfig {
		config := LoadObservabilityConfig()
		config.Traces.Enabled = false
		config.Logs.Enabled = false
		return config
	}),
)

// LogsOnlyModule provides only logging components
var LogsOnlyModule = fx.Module("observability-logs",
	fx.Provide(
		func() *ObservabilityConfig {
			config := LoadObservabilityConfig()
			config.Traces.Enabled = false
			config.Metrics.Enabled = false
			return config
		},
	),
	ProvideWithConfiguration(func() *ObservabilityConfig {
		config := LoadObservabilityConfig()
		config.Traces.Enabled = false
		config.Metrics.Enabled = false
		return config
	}),
)
