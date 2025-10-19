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

	// Admin Subscription Plans routes
	adminPlans := route.Group("/admin/plans")
	{
		adminPlans.POST("", protectFactory(subscriptionPlanUC.CreatePlan, roles.AdminRole))
		adminPlans.GET("", protectFactory(subscriptionPlanUC.ListAllPlans, roles.AdminRole))
		adminPlans.PUT("/:id", protectFactory(subscriptionPlanUC.UpdatePlan, roles.AdminRole))
		adminPlans.DELETE("/:id", protectFactory(subscriptionPlanUC.DeletePlan, roles.AdminRole))
	}

	// Subscriptions routes
	subscriptions := route.Group("/subscriptions")
	{
		// New subscription management endpoints
		subscriptions.POST("/subscribe", protectFactory(manageSubscriptionUC.SubscribeToPlan, roles.OwnerRole))
		subscriptions.DELETE("/cancel", protectFactory(manageSubscriptionUC.CancelSubscription, roles.OwnerRole))
		subscriptions.GET("/status", protectFactory(manageSubscriptionUC.GetSubscriptionStatus, roles.OwnerRole))

		// Existing routes
		subscriptions.GET("/plans", subscriptionUC.GetPlanFeatures)
		subscriptions.GET("/payment-history", protectFactory(subscriptionUC.GetPaymentHistory, roles.OwnerRole))
		subscriptions.POST("/cancel-old", protectFactory(subscriptionUC.CancelSubscription, roles.OwnerRole))
		subscriptions.POST("/change-plan", protectFactory(subscriptionUC.ChangePlan, roles.OwnerRole))
	}
}
