package company

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all company routes
func Routes(route *gin.RouterGroup, useCase usecases.ICompanyUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	companyRoutes := route.Group("/company")
	{
		// Platform Admin can create companies
		companyRoutes.POST("/", protectFactory(useCase.Create, roles.PlatformAdminRole))
		// Owner, OrgAdmin, and User can view company info
		companyRoutes.GET("/", protectFactory(useCase.Get, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Owner and OrgAdmin can update company info
		companyRoutes.PUT("/", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
	}
}
