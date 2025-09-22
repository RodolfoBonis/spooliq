package routes

import (
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/features/auth"
	auth_uc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/export"
	export_services "github.com/RodolfoBonis/spooliq/features/export/domain/services"
	"github.com/RodolfoBonis/spooliq/features/filaments"
	filaments_uc "github.com/RodolfoBonis/spooliq/features/filaments/domain/usecases"
	filament_metadata "github.com/RodolfoBonis/spooliq/features/filament-metadata"
	metadata_uc "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/presets"
	preset_services "github.com/RodolfoBonis/spooliq/features/presets/domain/services"
	"github.com/RodolfoBonis/spooliq/features/quotes"
	quote_uc "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/system"
	system_uc "github.com/RodolfoBonis/spooliq/features/system/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/users"
	user_services "github.com/RodolfoBonis/spooliq/features/users/domain/services"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitializeRoutes sets up all application routes.
func InitializeRoutes(
	router *gin.Engine,
	systemUc system_uc.SystemUseCase,
	authUc auth_uc.AuthUseCase,
	filamentsUc filaments_uc.FilamentUseCase,
	brandUc metadata_uc.BrandUseCase,
	materialUc metadata_uc.MaterialUseCase,
	quoteUc quote_uc.QuoteUseCase,
	userService user_services.UserService,
	presetService preset_services.PresetService,
	exportService export_services.ExportService,
	protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc,
	cacheMiddleware *middlewares.CacheMiddleware,
	logger logger.Logger,
) {

	root := router.Group("/v1")

	root.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	health.Routes(root, logger)
	auth.Routes(root, authUc, protectFactory)
	system.Routes(root, systemUc, cacheMiddleware)
	filaments.Routes(root, filamentsUc, protectFactory)
	filament_metadata.Routes(root, brandUc, materialUc, protectFactory)
	quotes.Routes(root, quoteUc, protectFactory)
	users.Routes(root, userService, protectFactory, logger)
	presets.Routes(root, presetService, protectFactory)
	export.Routes(root, exportService, protectFactory)
}
