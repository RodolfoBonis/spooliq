package subscriptions

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all subscription routes
func Routes(
	route *gin.RouterGroup,
	subscriptionUC usecases.ISubscriptionUseCase,
	protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc,
) {
	subscriptions := route.Group("/subscriptions")
	{
		// Public routes - anyone can view available plans
		subscriptions.GET("/plans", subscriptionUC.GetPlanFeatures)

		// Protected routes - Owner can manage subscriptions
		subscriptions.GET("/payment-history", protectFactory(subscriptionUC.GetPaymentHistory, roles.OwnerRole))
		subscriptions.POST("/cancel", protectFactory(subscriptionUC.CancelSubscription, roles.OwnerRole))
		subscriptions.POST("/change-plan", protectFactory(subscriptionUC.ChangePlan, roles.OwnerRole))
	}
}
