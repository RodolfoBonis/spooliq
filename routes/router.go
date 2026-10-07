package routes

import (
	"net/http"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/features/activity"
	activityuc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/admin"
	"github.com/RodolfoBonis/spooliq/features/auth"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/brand"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/budget"
	budgetuc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/company"
	companyuc "github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/customer"
	customeruc "github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/dashboard"
	"github.com/RodolfoBonis/spooliq/features/filament"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/material"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/model3d"
	model3duc "github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset"
	"github.com/RodolfoBonis/spooliq/features/profile"
	"github.com/RodolfoBonis/spooliq/features/slicer"
	sliceruc "github.com/RodolfoBonis/spooliq/features/slicer/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/stock"
	stockuc "github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/subscriptions"
	subscriptionuc "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/uploads"
	uploadsuc "github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/users"
	"github.com/RodolfoBonis/spooliq/features/webhooks"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitializeRoutes sets up all application routes.
func InitializeRoutes(
	router *gin.Engine,
	activityService activityuc.IActivityService,
	authUc authuc.AuthUseCase,
	registerUc *authuc.RegisterUseCase,
	brandUc branduc.IBrandUseCase,
	budgetUc budgetuc.IBudgetUseCase,
	publicBudgetUc budgetuc.IPublicBudgetUseCase,
	companyUc companyuc.ICompanyUseCase,
	brandingUc companyuc.IBrandingUseCase,
	subscriptionPaymentsUc companyuc.ISubscriptionPaymentsUseCase,
	customerUc customeruc.ICustomerUseCase,
	filamentUc filamentuc.IFilamentUseCase,
	stockUc stockuc.IStockUseCase,
	materialUc materialuc.IMaterialUseCase,
	model3dUc model3duc.IModel3DUseCase,
	slicerUc sliceruc.ISlicerUseCase,
	uploadsUc uploadsuc.IUploadUseCase,
	paymentMethodUc *subscriptionuc.PaymentMethodUseCase,
	subscriptionPlanUc *subscriptionuc.SubscriptionPlanUseCase,
	manageSubscriptionUc *subscriptionuc.ManageSubscriptionUseCase,
	presetHandler *preset.Handler,
	profileHandler *profile.Handler,
	dashboardHandler *dashboard.Handler,
	webhookHandler *webhooks.Handler,
	userHandler *users.Handler,
	adminHandler *admin.Handler,
	healthHandler *health.Handler,
	protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc,
	cacheMiddleware *middlewares.CacheMiddleware,
	logger logger.Logger,
) {
	// Unknown routes and methods return the standard error envelope instead of
	// gin's default plain-text 404/405.
	registerFallbacks(router)

	root := router.Group("/v1")

	root.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	healthHandler.Register(root)
	activity.Routes(root, activityService, protectFactory)
	auth.Routes(root, authUc, registerUc, protectFactory)
	brand.Routes(root, brandUc, protectFactory, cacheMiddleware)
	budget.Routes(root, budgetUc, protectFactory, cacheMiddleware)
	budget.PublicRoutes(root, publicBudgetUc)
	company.Routes(root, companyUc, brandingUc, subscriptionPaymentsUc, protectFactory)
	customer.Routes(root, customerUc, protectFactory)
	dashboard.SetupRoutes(root, dashboardHandler, protectFactory, cacheMiddleware)
	filament.Routes(root, filamentUc, protectFactory, cacheMiddleware)
	stock.Routes(root, stockUc, protectFactory, cacheMiddleware)
	material.Routes(root, materialUc, protectFactory, cacheMiddleware)
	model3d.Routes(root, model3dUc, protectFactory)
	slicer.Routes(root, slicerUc, protectFactory)
	preset.SetupRoutes(root, presetHandler, protectFactory)
	profile.Routes(root, profileHandler, protectFactory)
	uploads.Routes(root, uploadsUc, protectFactory)
	users.SetupRoutes(root, userHandler, protectFactory)
	webhooks.SetupRoutes(root, webhookHandler)
	admin.SetupRoutes(root, adminHandler, protectFactory)
	subscriptions.Routes(root, paymentMethodUc, subscriptionPlanUc, manageSubscriptionUc, protectFactory)
}

// registerFallbacks wires the no-route (404) and no-method (405) handlers so
// unmatched requests emit the standard error envelope. HandleMethodNotAllowed
// must be enabled on the engine (done at middleware setup) for NoMethod to run.
func registerFallbacks(router *gin.Engine) {
	router.NoRoute(func(c *gin.Context) {
		errors.AbortWith(c, errors.NotFoundErr(errors.CodeRouteNotFound, "Rota não encontrada"))
	})
	router.NoMethod(func(c *gin.Context) {
		errors.AbortWith(c, &errors.APIError{
			Status:  http.StatusMethodNotAllowed,
			Code:    errors.CodeMethodNotAllowed,
			Message: "Método não permitido",
		})
	})
}
