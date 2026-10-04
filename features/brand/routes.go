package brand

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
)

// cachePrefix is the cache/invalidation group for brand resources.
const cachePrefix = "brands"

// Cross-resource invalidation groups: filament responses embed the brand name and
// the dashboard aggregates by brand, so a brand mutation must clear those caches
// too, otherwise they would serve stale brand data.
var invalidateGroups = []string{cachePrefix, "filaments", "dashboard"}

// Routes configures all brand-related HTTP routes with authentication middleware and caching.
//
// Read endpoints wrap the handler with the cache (so it runs inside protectFactory,
// after auth) and write endpoints attach an invalidation middleware that clears this
// org's brand cache (plus the dependent filament and dashboard caches) after a
// successful mutation.
func Routes(route *gin.RouterGroup, useCase usecases.IBrandUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	brands := route.Group("/brands")
	{
		// All users can view brands
		brands.GET("", protectFactory(cacheMiddleware.Cache15Min(cachePrefix, useCase.FindAll), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		brands.GET("/:id", protectFactory(cacheMiddleware.Cache1Hour(cachePrefix, useCase.FindByID), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can create/update/delete brands
		brands.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		brands.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		brands.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
	}
}
