package material

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/gin-gonic/gin"
)

// cachePrefix is the cache/invalidation group for material resources.
const cachePrefix = "materials"

// Cross-resource invalidation groups: filament responses embed the material name
// and the dashboard aggregates by material, so a material mutation must clear
// those caches too, otherwise they would serve stale material data.
var invalidateGroups = []string{cachePrefix, "filaments", "dashboard"}

// Routes configures all material-related HTTP routes with authentication middleware and caching.
//
// Read endpoints wrap the handler with the cache (so it runs inside protectFactory,
// after auth) and write endpoints attach an invalidation middleware that clears this
// org's material cache (plus the dependent filament and dashboard caches) after a
// successful mutation.
func Routes(route *gin.RouterGroup, useCase usecases.IMaterialUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	materials := route.Group("/materials")
	{
		// All users can view materials
		materials.GET("", protectFactory(cacheMiddleware.Cache15Min(cachePrefix, useCase.FindAll), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		materials.GET("/:id", protectFactory(cacheMiddleware.Cache1Hour(cachePrefix, useCase.FindByID), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can create/update/delete materials
		materials.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		materials.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		materials.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
	}
}
