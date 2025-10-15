package budget

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all budget routes
func Routes(route *gin.RouterGroup, useCase usecases.IBudgetUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	budgetRoutes := route.Group("/budgets")
	{
		// All users can manage budgets
		budgetRoutes.POST("", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.PATCH("/:id/status", protectFactory(useCase.UpdateStatus, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.POST("/:id/duplicate", protectFactory(useCase.Duplicate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id/calculate", protectFactory(useCase.Recalculate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id/history", protectFactory(useCase.GetHistory, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/by-customer/:customer_id", protectFactory(useCase.FindByCustomer, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgetRoutes.GET("/:id/pdf", protectFactory(useCase.GeneratePDF, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can delete budgets
		budgetRoutes.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
