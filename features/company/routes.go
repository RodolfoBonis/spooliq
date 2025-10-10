package company

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all company routes
func Routes(route *gin.RouterGroup, useCase usecases.ICompanyUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	companyRoutes := route.Group("/company")
	{
		// All company routes require UserRole
		companyRoutes.POST("/", protectFactory(useCase.Create, roles.UserRole))
		companyRoutes.GET("/", protectFactory(useCase.Get, roles.UserRole))
		companyRoutes.PUT("/", protectFactory(useCase.Update, roles.UserRole))
	}
}
