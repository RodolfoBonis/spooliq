package subscriptions

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all subscription routes
func Routes(
	route *gin.RouterGroup,
	paymentMethodUC *usecases.PaymentMethodUseCase,
	subscriptionPlanUC *usecases.SubscriptionPlanUseCase,
	manageSubscriptionUC *usecases.ManageSubscriptionUseCase,
	protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc,
) {
	// Payment Methods routes
	paymentMethods := route.Group("/payment-methods")
	{
		paymentMethods.POST("", protectFactory(paymentMethodUC.AddPaymentMethod, roles.OwnerRole))
		paymentMethods.GET("", protectFactory(paymentMethodUC.ListPaymentMethods, roles.OwnerRole))
		paymentMethods.PUT("/:id/set-primary", protectFactory(paymentMethodUC.SetPrimaryPaymentMethod, roles.OwnerRole))
		paymentMethods.DELETE("/:id", protectFactory(paymentMethodUC.DeletePaymentMethod, roles.OwnerRole))
	}

	// Subscription Plans routes (public)
	route.GET("/plans", subscriptionPlanUC.ListActivePlans)

	// Subscriptions routes
	subscriptions := route.Group("/subscriptions")
	{
		subscriptions.POST("/subscribe", protectFactory(manageSubscriptionUC.SubscribeToPlan, roles.OwnerRole))
		subscriptions.DELETE("/cancel", protectFactory(manageSubscriptionUC.CancelSubscription, roles.OwnerRole))
		subscriptions.GET("/status", protectFactory(manageSubscriptionUC.GetSubscriptionStatus, roles.OwnerRole))
	}
}
