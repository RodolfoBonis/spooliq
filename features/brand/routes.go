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
		brands.GET("/", protectFactory(useCase.FindAll, roles.UserRole), cacheMiddleware.Cache15Min())
		brands.GET("/:id", protectFactory(useCase.FindByID, roles.UserRole), cacheMiddleware.Cache1Hour())
		brands.POST("/", protectFactory(useCase.Create, roles.OrgAdminRole))
		brands.PUT("/:id", protectFactory(useCase.Update, roles.OrgAdminRole))
		brands.DELETE("/:id", protectFactory(useCase.Delete, roles.OrgAdminRole))
	}
}
