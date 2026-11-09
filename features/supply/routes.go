package supply

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/usecases"
	"github.com/gin-gonic/gin"
)

// cachePrefix is the cache/invalidation group for supply resources.
const cachePrefix = "supplies"

// Cross-resource invalidation groups: budget item supplies snapshot the supply
// catalog and the dashboard aggregates supplies cost, so a supply mutation must
// clear those caches too, otherwise they would serve stale supply data.
var invalidateGroups = []string{cachePrefix, "budgets", "dashboard"}

// Routes configures all supply-related HTTP routes with authentication middleware and caching.
//
// Read endpoints wrap the handler with the cache (so it runs inside protectFactory,
// after auth) and write endpoints attach an invalidation middleware that clears this
// org's supply cache (plus the dependent budget and dashboard caches) after a
// successful mutation.
func Routes(route *gin.RouterGroup, useCase usecases.ISupplyUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	supplies := route.Group("/supplies")
	{
		// All users can view supplies
		supplies.GET("", protectFactory(cacheMiddleware.Cache15Min(cachePrefix, useCase.FindAll), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		supplies.GET("/:id", protectFactory(cacheMiddleware.Cache1Hour(cachePrefix, useCase.FindByID), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can create/update/delete supplies
		supplies.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		supplies.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		supplies.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
	}
}
