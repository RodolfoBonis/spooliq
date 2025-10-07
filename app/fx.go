package app

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/observability"
	"github.com/RodolfoBonis/spooliq/core/services"
	authDi "github.com/RodolfoBonis/spooliq/features/auth/di"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	brandDi "github.com/RodolfoBonis/spooliq/features/brand/di"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// NewFxApp cria e retorna uma nova instância da aplicação Fx.
func NewFxApp() *fx.App {
	return fx.New(
		logger.Module,
		config.Module,
		// Sistema completo de observabilidade OpenTelemetry/SignOz
		observability.Module,
		services.Module,
		middlewares.Module,
		authDi.AuthModule,
		brandDi.Module,
		fx.Provide(
			gin.New,
		),
		fx.Invoke(
			func(lc fx.Lifecycle, router *gin.Engine, authUc authuc.AuthUseCase, brandUc branduc.IBrandUseCase, monitoring *middlewares.MonitoringMiddleware, cacheMiddleware *middlewares.CacheMiddleware, obsManager *observability.Manager, helper *observability.Helper, redisService *services.RedisService, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, logger logger.Logger) {
				// Initialize Redis connection
				if err := redisService.Init(); err != nil {
					logger.Error(context.TODO(), "Failed to initialize Redis", map[string]interface{}{
						"error": err.Error(),
					})
				}

				// Setup middlewares and lifecycle hooks
				SetupMiddlewaresAndRoutes(lc, router, authUc, brandUc, protectFactory, logger, monitoring, obsManager, helper)
			},
		),
		// Incluir as migrações e seeds do init.go
		InitAndRun(),
	)
}
