package brand

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all brand-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IBrandUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	brands := route.Group("/brands")
	{
		// All users can view brands
		brands.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache15Min())
		brands.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache1Hour())
		// Only Owner and OrgAdmin can create/update/delete brands
		brands.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole))
		brands.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
		brands.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
