package app

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/observability"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	budgetuc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	companyuc "github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	customeruc "github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset"
	uploadsuc "github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/RodolfoBonis/spooliq/routes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// RegisterHooks registers application lifecycle hooks with legacy tracing.
// DEPRECATED: Use RegisterHooksWithObservability instead
func RegisterHooks(lifecycle fx.Lifecycle, router *gin.Engine, logger logger.Logger, monitoring *middlewares.MonitoringMiddleware, tracing *middlewares.TracingMiddleware) {
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				err := router.SetTrustedProxies([]string{})
				if err != nil {
					appError := errors.RootError(err.Error(), nil)
					logger.LogError(ctx, "Erro ao configurar trusted proxies", appError)
					panic(err)
				}
				config.SentryConfig()

				// Add tracing middleware first for OpenTelemetry
				router.Use(tracing.Middleware())
				router.Use(tracing.CustomTracing())

				router.Use(monitoring.SentryMiddleware())
				router.Use(monitoring.LogMiddleware)
				router.Use(gin.Logger())
				router.Use(gin.Recovery())
				router.Use(gin.ErrorLogger())
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info(ctx, "Stopping server.")
				return nil
			},
		},
	)
}

// RegisterHooksWithObservability registers application lifecycle hooks with new observability system.
func RegisterHooksWithObservability(lifecycle fx.Lifecycle, router *gin.Engine, logger logger.Logger, monitoring *middlewares.MonitoringMiddleware, obsManager *observability.Manager, helper *observability.Helper) {
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				err := router.SetTrustedProxies([]string{})
				if err != nil {
					appError := errors.RootError(err.Error(), nil)
					logger.LogError(ctx, "Erro ao configurar trusted proxies", appError)
					panic(err)
				}
				config.SentryConfig()

				// Add new observability middleware (automatic instrumentation)
				if obsManager.IsEnabled() {
					instrumentor := obsManager.GetInstrumentor()
					router.Use(instrumentor.InstrumentHTTPServer("spooliq-api"))
				}

				router.Use(monitoring.SentryMiddleware())
				router.Use(monitoring.LogMiddleware)
				router.Use(gin.Logger())
				router.Use(gin.Recovery())
				router.Use(gin.ErrorLogger())

				logger.Info(ctx, "Application started with enhanced observability", map[string]interface{}{
					"observability_enabled": obsManager.IsEnabled(),
					"auto_instrumentation":  obsManager.GetConfig().Features.AutoHTTP,
				})

				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info(ctx, "Stopping server.")
				return nil
			},
		},
	)
}

// SetupMiddlewaresAndRoutes configures middlewares BEFORE routes (critical for Gin)
func SetupMiddlewaresAndRoutes(lifecycle fx.Lifecycle, router *gin.Engine, authUc authuc.AuthUseCase, registerUc *authuc.RegisterUseCase, brandUc branduc.IBrandUseCase, budgetUc budgetuc.IBudgetUseCase, companyUc companyuc.ICompanyUseCase, customerUc customeruc.ICustomerUseCase, filamentUc filamentuc.IFilamentUseCase, materialUc materialuc.IMaterialUseCase, uploadsUc uploadsuc.IUploadUseCase, presetHandler *preset.Handler, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware, logger logger.Logger, monitoring *middlewares.MonitoringMiddleware, obsManager *observability.Manager, helper *observability.Helper) {
	// Configure trusted proxies
	err := router.SetTrustedProxies([]string{})
	if err != nil {
		appError := errors.RootError(err.Error(), nil)
		logger.LogError(context.Background(), "Erro ao configurar trusted proxies", appError)
		panic(err)
	}

	// Initialize Sentry
	config.SentryConfig()

	// CRITICAL: Register observability middleware FIRST (before any routes)
	if obsManager.IsEnabled() {
		instrumentor := obsManager.GetInstrumentor()
		router.Use(instrumentor.InstrumentHTTPServer("spooliq-api"))
		logger.Info(context.Background(), "Observability middleware registered", map[string]interface{}{
			"auto_instrumentation": obsManager.GetConfig().Features.AutoHTTP,
		})
	}

	// Register other middlewares
	router.Use(monitoring.SentryMiddleware())
	router.Use(monitoring.LogMiddleware)
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(gin.ErrorLogger())

	// Now register routes (AFTER all middlewares are set up)
	routes.InitializeRoutes(router, authUc, registerUc, brandUc, budgetUc, companyUc, customerUc, filamentUc, materialUc, uploadsUc, presetHandler, protectFactory, cacheMiddleware, logger)
	logger.Info(context.Background(), "Routes initialized after middleware setup")

	// Register lifecycle hooks for cleanup
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info(ctx, "Application started with enhanced observability", map[string]interface{}{
					"observability_enabled": obsManager.IsEnabled(),
					"auto_instrumentation":  obsManager.GetConfig().Features.AutoHTTP,
				})
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info(ctx, "Stopping server.")
				return nil
			},
		},
	)
}
