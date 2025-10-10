package budget

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all budget routes
func Routes(route *gin.RouterGroup, useCase usecases.IBudgetUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	budgetRoutes := route.Group("/budgets")
	{
		// All budget routes require UserRole
		budgetRoutes.POST("/", protectFactory(useCase.Create, roles.UserRole))
		budgetRoutes.GET("/", protectFactory(useCase.FindAll, roles.UserRole))
		budgetRoutes.GET("/:id", protectFactory(useCase.FindByID, roles.UserRole))
		budgetRoutes.PUT("/:id", protectFactory(useCase.Update, roles.UserRole))
		budgetRoutes.DELETE("/:id", protectFactory(useCase.Delete, roles.OrgAdmin))
		budgetRoutes.PATCH("/:id/status", protectFactory(useCase.UpdateStatus, roles.UserRole))
		budgetRoutes.POST("/:id/duplicate", protectFactory(useCase.Duplicate, roles.UserRole))
		budgetRoutes.GET("/:id/calculate", protectFactory(useCase.Recalculate, roles.UserRole))
		budgetRoutes.GET("/:id/history", protectFactory(useCase.GetHistory, roles.UserRole))
		budgetRoutes.GET("/by-customer/:customer_id", protectFactory(useCase.FindByCustomer, roles.UserRole))
		budgetRoutes.GET("/:id/pdf", protectFactory(useCase.GeneratePDF, roles.UserRole))
	}
}
