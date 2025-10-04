package app

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
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

// RegisterHooksWithObservability has been temporarily removed while we fix observability implementation
// Use RegisterHooks instead for now
