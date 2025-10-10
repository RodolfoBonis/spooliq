package routes

import (
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
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
	"github.com/RodolfoBonis/spooliq/features/filament"
	filamentuc "github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/material"
	materialuc "github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset"
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
	authUc authuc.AuthUseCase,
	registerUc *authuc.RegisterUseCase,
	brandUc branduc.IBrandUseCase,
	budgetUc budgetuc.IBudgetUseCase,
	companyUc companyuc.ICompanyUseCase,
	customerUc customeruc.ICustomerUseCase,
	filamentUc filamentuc.IFilamentUseCase,
	materialUc materialuc.IMaterialUseCase,
	uploadsUc uploadsuc.IUploadUseCase,
	presetHandler *preset.Handler,
	webhookHandler *webhooks.Handler,
	userHandler *users.Handler,
	adminHandler *admin.Handler,
	protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc,
	cacheMiddleware *middlewares.CacheMiddleware,
	logger logger.Logger,
) {

	root := router.Group("/v1")

	root.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	health.Routes(root, logger)
	auth.Routes(root, authUc, registerUc, protectFactory)
	brand.Routes(root, brandUc, protectFactory, cacheMiddleware)
	budget.Routes(root, budgetUc, protectFactory)
	company.Routes(root, companyUc, protectFactory)
	customer.Routes(root, customerUc, protectFactory)
	filament.Routes(root, filamentUc, protectFactory, cacheMiddleware)
	material.Routes(root, materialUc, protectFactory, cacheMiddleware)
	preset.SetupRoutes(root, presetHandler, protectFactory)
	uploads.Routes(root, uploadsUc, protectFactory)
	users.SetupRoutes(root, userHandler, protectFactory)
	webhooks.SetupRoutes(root, webhookHandler)
	admin.SetupRoutes(root, adminHandler, protectFactory)
}
