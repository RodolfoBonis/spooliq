package dashboard

import (
	"time"

	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the dashboard routes.
//
// Dashboard responses are per-user and vary by query; the cache wraps each handler
// so it runs inside protectFactory (after auth, with organization_id in context).
// Dashboard entries are not actively invalidated and rely on their short TTLs.
func SetupRoutes(router *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	dashboard := router.Group("/dashboard")

	// Cache configs for dashboard endpoints. Organization ID and the full query
	// string are always part of the key; VaryByUser adds the user ID on top.
	dashboardCache := middlewares.CacheConfig{
		TTL:        5 * time.Minute,
		VaryByUser: true,
		KeyPrefix:  "dashboard",
	}
	recentActivityCache := middlewares.CacheConfig{
		TTL:        2 * time.Minute,
		VaryByUser: true,
		KeyPrefix:  "dashboard:activity",
	}

	{
		dashboard.GET("/overview", protectFactory(cacheMiddleware.Wrap(handler.GetOverview, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/revenue-trend", protectFactory(cacheMiddleware.Wrap(handler.GetRevenueTrend, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/conversion-funnel", protectFactory(cacheMiddleware.Wrap(handler.GetConversionFunnel, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/recent-activity", protectFactory(cacheMiddleware.Wrap(handler.GetRecentActivity, recentActivityCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/top-customers", protectFactory(cacheMiddleware.Wrap(handler.GetTopCustomers, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/operational-insights", protectFactory(cacheMiddleware.Wrap(handler.GetOperationalInsights, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/top-filaments", protectFactory(cacheMiddleware.Wrap(handler.GetTopFilaments, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/top-materials", protectFactory(cacheMiddleware.Wrap(handler.GetTopMaterials, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		dashboard.GET("/goals-alerts", protectFactory(cacheMiddleware.Wrap(handler.GetGoalsAlerts, dashboardCache), roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}
}
