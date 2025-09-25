package routes

import (
	"github.com/RodolfoBonis/spooliq/core/health"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/auth"
	authuc "github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	branduc "github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"

	"github.com/RodolfoBonis/spooliq/features/brand"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitializeRoutes sets up all application routes.
func InitializeRoutes(
	router *gin.Engine,
	authUc authuc.AuthUseCase,
	brandUc branduc.IBrandUseCase,
	protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc,
	logger logger.Logger,
) {

	root := router.Group("/v1")

	root.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	health.Routes(root, logger)
	auth.Routes(root, authUc, protectFactory)
	brand.Routes(root, brandUc, protectFactory)
}
