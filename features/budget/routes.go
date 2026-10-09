package budget

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/gin-gonic/gin"
)

// statusInvalidateGroups are the cache groups a status change can make stale:
// completing a budget auto-deducts filament stock (filaments) and moves the
// dashboard aggregates.
var statusInvalidateGroups = []string{"filaments", "dashboard"}

// dashboardInvalidateGroups covers mutations that change dashboard aggregates
// (counts, revenue, profit) without touching stock.
var dashboardInvalidateGroups = []string{"dashboard"}

// Routes registers all budget routes
func Routes(route *gin.RouterGroup, useCase usecases.IBudgetUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	budgetRoutes := route.Group("/budgets")
	{
		// All users can manage budgets
		budgetRoutes.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(dashboardInvalidateGroups...))
		// Stateless cost preview (never persists). Same roles as create.
		budgetRoutes.POST("/preview", protectFactory(useCase.Preview, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/export.csv", protectFactory(useCase.ExportCSV, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(dashboardInvalidateGroups...))
		budgetRoutes.PATCH("/:id/status", protectFactory(useCase.UpdateStatus, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(statusInvalidateGroups...))
		budgetRoutes.POST("/:id/duplicate", protectFactory(useCase.Duplicate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(dashboardInvalidateGroups...))
		// Public share link management (create/return token, revoke).
		budgetRoutes.POST("/:id/share", protectFactory(useCase.Share, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.DELETE("/:id/share", protectFactory(useCase.RevokeShare, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Recalculate (mutates stored costs) is a POST and only allowed for drafts.
		budgetRoutes.POST("/:id/recalculate", protectFactory(useCase.Recalculate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole), cacheMiddleware.InvalidateMiddleware(dashboardInvalidateGroups...))
		// Deprecated: read-only view of stored costs (kept for backward compatibility).
		budgetRoutes.GET("/:id/calculate", protectFactory(useCase.GetCalculation, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id/history", protectFactory(useCase.GetHistory, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/by-customer/:customer_id", protectFactory(useCase.FindByCustomer, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id/pdf", protectFactory(useCase.GeneratePDF, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can delete budgets
		budgetRoutes.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole), cacheMiddleware.InvalidateMiddleware(dashboardInvalidateGroups...))
	}
}

// PublicRoutes registers the customer-facing public budget routes. These are mounted
// WITHOUT the auth or subscription middleware: the share token is the only
// credential, and abuse is mitigated by a per-IP rate limit inside the use case.
func PublicRoutes(route *gin.RouterGroup, publicUseCase usecases.IPublicBudgetUseCase) {
	public := route.Group("/public/budgets")
	{
		public.GET("/:token", publicUseCase.GetByToken)
		public.GET("/:token/pdf", publicUseCase.GetPDF)
		public.POST("/:token/approve", publicUseCase.Approve)
		public.POST("/:token/reject", publicUseCase.Reject)
	}
}
