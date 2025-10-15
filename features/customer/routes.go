package customer

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all customer routes
func Routes(route *gin.RouterGroup, useCase usecases.ICustomerUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	customerRoutes := route.Group("/customers")
	{
		// All users can manage customers
		customerRoutes.POST("/", protectFactory(useCase.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		customerRoutes.GET("/", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		customerRoutes.GET("/search", protectFactory(useCase.Search, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		customerRoutes.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		customerRoutes.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Only Owner and OrgAdmin can delete customers
		customerRoutes.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
