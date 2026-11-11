// Package stock wires the filament stock-movement HTTP routes.
package stock

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
	"github.com/gin-gonic/gin"
)

// invalidateGroups are the cache groups a stock movement must clear: the filament
// cache (list/search/detail reflect stock_grams and is_low_stock) and the dashboard
// cache (the low-stock widget aggregates filament stock).
var invalidateGroups = []string{"filaments", "dashboard"}

// Routes registers the filament stock movement endpoints under /filaments/:id.
//
// Read endpoints are not cached (movements and balances change frequently and must
// be fresh). The write endpoint invalidates the filament and dashboard caches so a
// subsequent filament/dashboard read never serves a stale balance.
func Routes(route *gin.RouterGroup, useCase usecases.IStockUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	filaments := route.Group("/filaments")
	{
		// All roles may record movements and read the ledger.
		filaments.POST("/:id/stock-movements", protectFactory(useCase.CreateMovement, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(invalidateGroups...))
		filaments.GET("/:id/stock-movements", protectFactory(useCase.ListMovements, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}
}
