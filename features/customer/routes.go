package customer

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all customer routes
func Routes(route *gin.RouterGroup, useCase usecases.ICustomerUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	customerRoutes := route.Group("/customers")
	{
		// All customer routes require UserRole
		customerRoutes.POST("/", protectFactory(useCase.Create, roles.UserRole))
		customerRoutes.GET("/", protectFactory(useCase.FindAll, roles.UserRole))
		customerRoutes.GET("/search", protectFactory(useCase.Search, roles.UserRole))
		customerRoutes.GET("/:id", protectFactory(useCase.FindByID, roles.UserRole))
		customerRoutes.PUT("/:id", protectFactory(useCase.Update, roles.UserRole))
		customerRoutes.DELETE("/:id", protectFactory(useCase.Delete, roles.UserRole))
	}
}
