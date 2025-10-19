package company

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/company/domain/usecases"
	subscriptionUsecases "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all company routes
func Routes(route *gin.RouterGroup, useCase usecases.ICompanyUseCase, brandingUseCase usecases.IBrandingUseCase, subscriptionUseCase subscriptionUsecases.ISubscriptionUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	companyRoutes := route.Group("/company")
	{
		// Platform Admin can create companies
		companyRoutes.POST("/", protectFactory(useCase.Create, roles.PlatformAdminRole))
		// Owner, OrgAdmin, and User can view company info
		companyRoutes.GET("/", protectFactory(useCase.Get, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Owner and OrgAdmin can update company info
		companyRoutes.PUT("/", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
		// Owner and OrgAdmin can upload logo
		companyRoutes.POST("/logo", protectFactory(useCase.UploadLogo, roles.OwnerRole, roles.OrgAdminRole))

		// Branding endpoints
		// All authenticated users can view branding
		companyRoutes.GET("/branding", protectFactory(brandingUseCase.GetBranding, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		// Owner and OrgAdmin can update branding
		companyRoutes.PUT("/branding", protectFactory(brandingUseCase.UpdateBranding, roles.OwnerRole, roles.OrgAdminRole))
		// All authenticated users can list templates
		companyRoutes.GET("/branding/templates", protectFactory(brandingUseCase.ListTemplates, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Subscription endpoints
		// Owner can view payment history
		companyRoutes.GET("/subscription/payments", protectFactory(subscriptionUseCase.GetPaymentHistory, roles.OwnerRole))
	}
}
