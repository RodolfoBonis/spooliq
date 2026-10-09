package app

import (
	"context"
	"database/sql"
	"time"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/fxmodule"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/account"
	accountDi "github.com/RodolfoBonis/spooliq/features/account/di"
	activityDi "github.com/RodolfoBonis/spooliq/features/activity/di"
	activityuc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
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
	"github.com/RodolfoBonis/spooliq/features/dashboard"
	filamentDi "github.com/RodolfoBonis/spooliq/features/filament/di"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	materialDi "github.com/RodolfoBonis/spooliq/features/material/di"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	model3dDi "github.com/RodolfoBonis/spooliq/features/model3d/di"
	model3duc "github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset"
	"github.com/RodolfoBonis/spooliq/features/profile"
	profileDi "github.com/RodolfoBonis/spooliq/features/profile/di"
	slicerDi "github.com/RodolfoBonis/spooliq/features/slicer/di"
	sliceruc "github.com/RodolfoBonis/spooliq/features/slicer/domain/usecases"
	stockDi "github.com/RodolfoBonis/spooliq/features/stock/di"
	stockuc "github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
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
	"gorm.io/gorm"
)

// NewFxApp creates and returns a new FX application instance.
func NewFxApp() *fx.App {
	return fx.New(
		fx.StartTimeout(60*time.Second),
		fx.StopTimeout(30*time.Second),
		config.Module,
		fxmodule.ProvideWithConfiguration(
			otelagent.WithServiceName("spooliq-api"),
			otelagent.WithServiceNamespace("spooliq"),
			otelagent.WithServiceVersion("1.0.0"),
		),
		services.Module,
		middlewares.Module,
		activityDi.Module,
		authDi.AuthModule,
		brandDi.Module,
		budgetDi.Module,
		companyDi.Module,
		customerDi.Module,
		filamentDi.Module,
		materialDi.Module,
		model3dDi.Module,
		slicerDi.Module,
		stockDi.Module,
		dashboard.Module,
		preset.Module,
		profileDi.Module,
		uploadsDi.Module,
		usersDi.UsersModule,
		accountDi.AccountModule,
		webhookDi.Module,
		adminDi.AdminModule,
		subscriptionsDi.Module,
		fx.Provide(
			gin.New,
			func(logger logger.Logger) *services.CDNService {
				return services.NewCDNService(
					config.EnvCDNBaseURL(),
					config.EnvCDNKeys(),
					logger,
				)
			},
			func(cdnService *services.CDNService, logger logger.Logger) *services.PDFService {
				return services.NewPDFService(cdnService, logger)
			},
			func(logger logger.Logger) *services.ThumbnailService {
				return services.NewThumbnailService(logger)
			},
			func(db *gorm.DB, redisService *services.RedisService, log logger.Logger) *health.Handler {
				var sqlDB *sql.DB
				if db != nil {
					sqlDB, _ = db.DB()
				}
				// Resolve lazily: Redis is initialized later, in fx.Invoke.
				return health.NewHandler(sqlDB, redisService.GetClient, log)
			},
		),
		fx.Invoke(
			func(lc fx.Lifecycle, router *gin.Engine, activityService activityuc.IActivityService, authUc authuc.AuthUseCase, registerUc *authuc.RegisterUseCase, brandUc branduc.IBrandUseCase, budgetUc budgetuc.IBudgetUseCase, publicBudgetUc budgetuc.IPublicBudgetUseCase, companyUc companyuc.ICompanyUseCase, brandingUc companyuc.IBrandingUseCase, subscriptionPaymentsUc companyuc.ISubscriptionPaymentsUseCase, customerUc customeruc.ICustomerUseCase, filamentUc filamentuc.IFilamentUseCase, stockUc stockuc.IStockUseCase, materialUc materialuc.IMaterialUseCase, model3dUc model3duc.IModel3DUseCase, slicerUc sliceruc.ISlicerUseCase, uploadsUc uploadsuc.IUploadUseCase, paymentMethodUc *subscriptionuc.PaymentMethodUseCase, subscriptionPlanUc *subscriptionuc.SubscriptionPlanUseCase, manageSubscriptionUc *subscriptionuc.ManageSubscriptionUseCase, presetHandler *preset.Handler, profileHandler *profile.Handler, dashboardHandler *dashboard.Handler, webhookHandler *webhooks.Handler, userHandler *users.Handler, accountHandler *account.Handler, adminHandler *admin.Handler, healthHandler *health.Handler, monitoring *middlewares.MonitoringMiddleware, cacheMiddleware *middlewares.CacheMiddleware, subscriptionMiddleware *middlewares.SubscriptionMiddleware, agent *otelagent.Agent, redisService *services.RedisService, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, logger logger.Logger) {
				// Initialize Redis connection
				if err := redisService.Init(); err != nil {
					logger.Error(context.TODO(), "Failed to initialize Redis", map[string]interface{}{
						"error": err.Error(),
					})
				}

				// Setup middlewares and lifecycle hooks
				SetupMiddlewaresAndRoutes(lc, router, activityService, authUc, registerUc, brandUc, budgetUc, publicBudgetUc, companyUc, brandingUc, subscriptionPaymentsUc, customerUc, filamentUc, stockUc, materialUc, model3dUc, slicerUc, uploadsUc, paymentMethodUc, subscriptionPlanUc, manageSubscriptionUc, presetHandler, profileHandler, dashboardHandler, webhookHandler, userHandler, accountHandler, adminHandler, healthHandler, protectFactory, cacheMiddleware, subscriptionMiddleware, logger, monitoring, agent)
			},
		),
		InitAndRun(),
	)
}
