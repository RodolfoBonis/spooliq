package dashboard

import (
	"time"

	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the dashboard routes.
func SetupRoutes(router *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	dashboard := router.Group("/dashboard")

	// Cache configs for dashboard endpoints
	dashboardCache := middlewares.CacheConfig{
		TTL:         5 * time.Minute,
		VaryByUser:  true,
		VaryByQuery: true,
		KeyPrefix:   "dashboard",
	}
	recentActivityCache := middlewares.CacheConfig{
		TTL:         2 * time.Minute,
		VaryByUser:  true,
		VaryByQuery: true,
		KeyPrefix:   "dashboard:activity",
	}

	{
		dashboard.GET("/overview", protectFactory(handler.GetOverview, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/revenue-trend", protectFactory(handler.GetRevenueTrend, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/conversion-funnel", protectFactory(handler.GetConversionFunnel, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/recent-activity", protectFactory(handler.GetRecentActivity, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(recentActivityCache))
		dashboard.GET("/top-customers", protectFactory(handler.GetTopCustomers, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/operational-insights", protectFactory(handler.GetOperationalInsights, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/top-filaments", protectFactory(handler.GetTopFilaments, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/top-materials", protectFactory(handler.GetTopMaterials, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
		dashboard.GET("/goals-alerts", protectFactory(handler.GetGoalsAlerts, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.Cache(dashboardCache))
	}
}
