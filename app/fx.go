package app

import (
	"context"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/observability"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/admin"
	adminDi "github.com/RodolfoBonis/spooliq/features/admin/di"
	authDi "github.com/RodolfoBonis/spooliq/features/auth/di"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	brandDi "github.com/RodolfoBonis/spooliq/features/brand/di"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	budgetDi "github.com/RodolfoBonis/spooliq/features/budget/di"
	budgetuc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	companyDi "github.com/RodolfoBonis/spooliq/features/company/di"
	companyuc "github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	customerDi "github.com/RodolfoBonis/spooliq/features/customer/di"
	customeruc "github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	filamentDi "github.com/RodolfoBonis/spooliq/features/filament/di"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	materialDi "github.com/RodolfoBonis/spooliq/features/material/di"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset"
	subscriptionsDi "github.com/RodolfoBonis/spooliq/features/subscriptions/di"
	subscriptionuc "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	uploadsDi "github.com/RodolfoBonis/spooliq/features/uploads/di"
	uploadsuc "github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/users"
	usersDi "github.com/RodolfoBonis/spooliq/features/users/di"
	"github.com/RodolfoBonis/spooliq/features/webhooks"
	webhookDi "github.com/RodolfoBonis/spooliq/features/webhooks/di"
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
		budgetDi.Module,
		companyDi.Module,
		customerDi.Module,
		filamentDi.Module,
		materialDi.Module,
		preset.Module,
		uploadsDi.Module,
		usersDi.UsersModule,
		webhookDi.Module,
		adminDi.AdminModule,
		subscriptionsDi.Module,
		fx.Provide(
			gin.New,
			func(logger logger.Logger) *services.CDNService {
				return services.NewCDNService(
					config.EnvCDNBaseURL(),
					config.EnvCDNAPIKey(),
					config.EnvCDNBucket(),
					logger,
				)
			},
			func(cdnService *services.CDNService, logger logger.Logger) *services.PDFService {
				return services.NewPDFService(cdnService, logger)
			},
		),
		fx.Invoke(
			func(lc fx.Lifecycle, router *gin.Engine, authUc authuc.AuthUseCase, registerUc *authuc.RegisterUseCase, brandUc branduc.IBrandUseCase, budgetUc budgetuc.IBudgetUseCase, companyUc companyuc.ICompanyUseCase, brandingUc companyuc.IBrandingUseCase, customerUc customeruc.ICustomerUseCase, filamentUc filamentuc.IFilamentUseCase, materialUc materialuc.IMaterialUseCase, uploadsUc uploadsuc.IUploadUseCase, subscriptionUc subscriptionuc.ISubscriptionUseCase, presetHandler *preset.Handler, webhookHandler *webhooks.Handler, userHandler *users.Handler, adminHandler *admin.Handler, monitoring *middlewares.MonitoringMiddleware, cacheMiddleware *middlewares.CacheMiddleware, subscriptionMiddleware *middlewares.SubscriptionMiddleware, obsManager *observability.Manager, helper *observability.Helper, redisService *services.RedisService, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, logger logger.Logger) {
				// Initialize Redis connection
				if err := redisService.Init(); err != nil {
					logger.Error(context.TODO(), "Failed to initialize Redis", map[string]interface{}{
						"error": err.Error(),
					})
				}

				// Setup middlewares and lifecycle hooks
				SetupMiddlewaresAndRoutes(lc, router, authUc, registerUc, brandUc, budgetUc, companyUc, brandingUc, customerUc, filamentUc, materialUc, uploadsUc, subscriptionUc, presetHandler, webhookHandler, userHandler, adminHandler, protectFactory, cacheMiddleware, subscriptionMiddleware, logger, monitoring, obsManager, helper)
			},
		),
		// Incluir as migrações e seeds do init.go
		InitAndRun(),
	)
}
