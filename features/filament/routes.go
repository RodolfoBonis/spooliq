package filament

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"github.com/gin-gonic/gin"
)

// cachePrefix is the cache/invalidation group for filament resources.
const cachePrefix = "filaments"

// Cross-resource invalidation groups: the dashboard aggregates filament usage, so
// a filament mutation must clear the dashboard cache too, otherwise it would serve
// stale figures.
var invalidateGroups = []string{cachePrefix, "dashboard"}

// Routes configures all filament-related HTTP routes with authentication middleware and caching.
//
// Read endpoints wrap the handler with the cache (so it runs inside protectFactory,
// after auth) and write endpoints attach an invalidation middleware that clears this
// org's filament cache (plus the dependent dashboard cache) after a successful
// mutation.
func Routes(route *gin.RouterGroup, useCase usecases.IFilamentUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	filaments := route.Group("/filaments")
	{
		// All users can view and search filaments
		filaments.GET("", protectFactory(cacheMiddleware.Cache5Min(cachePrefix, useCase.FindAll), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		filaments.GET("/search", protectFactory(cacheMiddleware.Cache5Min(cachePrefix, useCase.Search), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		filaments.GET("/:id", protectFactory(cacheMiddleware.Cache15Min(cachePrefix, useCase.FindByID), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// All users can create/update filaments
		filaments.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		filaments.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		// Only Owner and OrgAdmin can delete filaments
		filaments.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
	}
}
