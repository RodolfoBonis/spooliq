package app

import (
	"context"
	"fmt"

	otelagent "github.com/RodolfoBonis/go-otel-agent"
	"github.com/RodolfoBonis/go-otel-agent/integration/ginmiddleware"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/account"
	activityuc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/admin"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	budgetuc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	companyuc "github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	customeruc "github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/dashboard"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	model3duc "github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	notificationuc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments"
	"github.com/RodolfoBonis/spooliq/features/preset"
	"github.com/RodolfoBonis/spooliq/features/profile"
	sliceruc "github.com/RodolfoBonis/spooliq/features/slicer/domain/usecases"
	stockuc "github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
	subscriptionuc "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	supplyuc "github.com/RodolfoBonis/spooliq/features/supply/domain/usecases"
	uploadsuc "github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/users"
	"github.com/RodolfoBonis/spooliq/features/webhooks"
	"github.com/RodolfoBonis/spooliq/routes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// SetupMiddlewaresAndRoutes configures middlewares BEFORE routes (critical for Gin)
func SetupMiddlewaresAndRoutes(lifecycle fx.Lifecycle, router *gin.Engine, activityService activityuc.IActivityService, notificationService notificationuc.INotificationService, authUc authuc.AuthUseCase, registerUc *authuc.RegisterUseCase, brandUc branduc.IBrandUseCase, budgetUc budgetuc.IBudgetUseCase, publicBudgetUc budgetuc.IPublicBudgetUseCase, companyUc companyuc.ICompanyUseCase, brandingUc companyuc.IBrandingUseCase, subscriptionPaymentsUc companyuc.ISubscriptionPaymentsUseCase, customerUc customeruc.ICustomerUseCase, filamentUc filamentuc.IFilamentUseCase, stockUc stockuc.IStockUseCase, materialUc materialuc.IMaterialUseCase, supplyUc supplyuc.ISupplyUseCase, model3dUc model3duc.IModel3DUseCase, slicerUc sliceruc.ISlicerUseCase, uploadsUc uploadsuc.IUploadUseCase, paymentMethodUc *subscriptionuc.PaymentMethodUseCase, subscriptionPlanUc *subscriptionuc.SubscriptionPlanUseCase, manageSubscriptionUc *subscriptionuc.ManageSubscriptionUseCase, presetHandler *preset.Handler, profileHandler *profile.Handler, dashboardHandler *dashboard.Handler, webhookHandler *webhooks.Handler, paymentsHandler *payments.Handler, userHandler *users.Handler, accountHandler *account.Handler, adminHandler *admin.Handler, healthHandler *health.Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware, subscriptionMiddleware *middlewares.SubscriptionMiddleware, logger logger.Logger, monitoring *middlewares.MonitoringMiddleware, agent *otelagent.Agent) {
	// Configure trusted proxies
	err := router.SetTrustedProxies([]string{})
	if err != nil {
		appError := errors.RootError(err.Error(), nil)
		logger.LogError(context.Background(), "Erro ao configurar trusted proxies", appError)
		panic(err)
	}

	router.MaxMultipartMemory = 32 << 20 // 32MB

	// 405 responses require this flag; without it gin treats a known path with
	// an unknown method as a 404.
	router.HandleMethodNotAllowed = true

	// Wire the package-level error logger so Respond can record internal (500)
	// errors, and install the shared validator as gin's binding validator.
	errors.SetLogger(logger)
	validation.Register()

	config.SentryConfig()

	// CORS must run first so preflight (OPTIONS) requests are answered before
	// routing; without it they fall through to 404.
	router.Use(middlewares.Cors())

	// Sanitize secret path tokens (public share token, webhook token) for tracing and
	// metrics BEFORE the observability middleware records the span name and url.path.
	// Routing has already matched and handlers read path params (not URL.Path), so this
	// only changes what is logged/traced, never behavior.
	router.Use(func(c *gin.Context) {
		if masked := helpers.MaskSensitivePath(c.Request.URL.Path); masked != c.Request.URL.Path {
			c.Request.URL.Path = masked
		}
		c.Next()
	})

	// Observability middleware (tracing + metrics + enrichment)
	router.Use(ginmiddleware.New(agent, "spooliq-api"))

	// Recovery wraps Sentry so sentrygin (Repanic) captures the panic first and
	// re-panics into Recovery, which logs it and returns the 500 envelope.
	router.Use(middlewares.Recovery(logger))
	router.Use(monitoring.SentryMiddleware())
	router.Use(monitoring.LogMiddleware)
	// Mask every secret path token in the access log so it never reaches stdout/log
	// sinks: the public share token and the InfinitePay webhook token. Everything
	// else mirrors gin's default format.
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			helpers.MaskSensitivePath(param.Path),
		)
	}))
	router.Use(gin.ErrorLogger())

	routes.InitializeRoutes(router, activityService, notificationService, authUc, registerUc, brandUc, budgetUc, publicBudgetUc, companyUc, brandingUc, subscriptionPaymentsUc, customerUc, filamentUc, stockUc, materialUc, supplyUc, model3dUc, slicerUc, uploadsUc, paymentMethodUc, subscriptionPlanUc, manageSubscriptionUc, presetHandler, profileHandler, dashboardHandler, webhookHandler, paymentsHandler, userHandler, accountHandler, adminHandler, healthHandler, protectFactory, cacheMiddleware, logger)
	logger.Info(context.Background(), "Routes initialized after middleware setup")

	// Register lifecycle hooks for cleanup
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				logger.Info(ctx, "Application started")
				return nil
			},
			OnStop: func(ctx context.Context) error {
				logger.Info(ctx, "Stopping server.")
				return nil
			},
		},
	)
}
