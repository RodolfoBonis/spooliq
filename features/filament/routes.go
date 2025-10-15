package filament

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all filament-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IFilamentUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	filaments := route.Group("/filaments")
	{
		// All users can view and search filaments
		filaments.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache5Min())
		filaments.GET("/search", protectFactory(useCase.Search, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache5Min())
		filaments.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache15Min())
		// All users can create/update filaments
		filaments.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		filaments.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can delete filaments
		filaments.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
