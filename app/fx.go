package app

import (
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/auth/di"
	auth_uc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	exportdi "github.com/RodolfoBonis/spooliq/features/export/di"
	filamentsdi "github.com/RodolfoBonis/spooliq/features/filaments/di"
	filaments_uc "github.com/RodolfoBonis/spooliq/features/filaments/domain/usecases"
	presetsdi "github.com/RodolfoBonis/spooliq/features/presets/di"
	quotesdi "github.com/RodolfoBonis/spooliq/features/quotes/di"
	systemdi "github.com/RodolfoBonis/spooliq/features/system/di"
	system_uc "github.com/RodolfoBonis/spooliq/features/system/domain/usecases"
	usersdi "github.com/RodolfoBonis/spooliq/features/users/di"
	"github.com/RodolfoBonis/spooliq/routes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// NewFxApp cria e retorna uma nova instância da aplicação Fx.
func NewFxApp() *fx.App {
	return fx.New(
		logger.Module,
		config.Module,
		services.Module,
		middlewares.Module,
		di.AuthModule,
		systemdi.SystemModule,
		filamentsdi.FilamentsModule,
		quotesdi.QuotesModule,
		exportdi.Module,
		usersdi.Module,
		presetsdi.Module,
		fx.Provide(
			gin.New,
		),
		fx.Invoke(
			func(lc fx.Lifecycle, router *gin.Engine, systemUc system_uc.SystemUseCase, authUc auth_uc.AuthUseCase, filamentsUc filaments_uc.FilamentUseCase, monitoring *middlewares.MonitoringMiddleware, cacheMiddleware *middlewares.CacheMiddleware, redisService *services.RedisService, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, logger logger.Logger) {
				// Initialize Redis connection
				if err := redisService.Init(); err != nil {
					logger.Error(nil, "Failed to initialize Redis", map[string]interface{}{
						"error": err.Error(),
					})
				}

				routes.InitializeRoutes(router, systemUc, authUc, filamentsUc, protectFactory, cacheMiddleware, logger)
				RegisterHooks(lc, router, logger, monitoring)
				_ = services.OpenConnection(logger)
			},
		),
		// Incluir as migrações e seeds do init.go
		InitAndRun(),
	)
}
