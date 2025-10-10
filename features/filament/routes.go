package filament

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all filament-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IFilamentUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	filaments := route.Group("/filaments")
	{
		filaments.GET("/", protectFactory(useCase.FindAll, roles.UserRole), cacheMiddleware.Cache5Min())
		filaments.GET("/search", protectFactory(useCase.Search, roles.UserRole), cacheMiddleware.Cache5Min())
		filaments.GET("/:id", protectFactory(useCase.FindByID, roles.UserRole), cacheMiddleware.Cache15Min())
		filaments.POST("/", protectFactory(useCase.Create, roles.UserRole))
		filaments.PUT("/:id", protectFactory(useCase.Update, roles.UserRole))
		filaments.DELETE("/:id", protectFactory(useCase.Delete, roles.UserRole))
	}
}
