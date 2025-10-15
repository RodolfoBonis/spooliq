package material

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all material-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IMaterialUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	materials := route.Group("/materials")
	{
		// All users can view materials
		materials.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache15Min())
		materials.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache1Hour())
		// Only Owner and OrgAdmin can create/update/delete materials
		materials.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole))
		materials.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
		materials.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
