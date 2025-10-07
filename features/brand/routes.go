package brand

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all brand-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IBrandUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	brands := route.Group("/brands")
	{
		brands.GET("/", cacheMiddleware.Cache15Min(), protectFactory(useCase.FindAll, roles.AdminRole))
		brands.GET("/:id", cacheMiddleware.Cache1Hour(), protectFactory(useCase.FindByID, roles.AdminRole))
		brands.POST("/", protectFactory(useCase.Create, roles.AdminRole))
		brands.PUT("/:id", protectFactory(useCase.Update, roles.AdminRole))
	}
}
