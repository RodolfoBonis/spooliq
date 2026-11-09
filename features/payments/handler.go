// Package payments wires the InfinitePay payment feature's HTTP surface: the
// authenticated company integration endpoints, the authenticated budget payment
// endpoints, the public (no-auth) pay/confirm endpoints and the webhook.
package payments

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Handler bundles the payment use cases for route registration.
type Handler struct {
	integration    *usecases.IntegrationUseCase
	budgetPayments *usecases.BudgetPaymentsUseCase
	public         *usecases.PublicPaymentUseCase
	webhook        *usecases.WebhookUseCase
}

// NewHandler builds the payments handler.
func NewHandler(
	integration *usecases.IntegrationUseCase,
	budgetPayments *usecases.BudgetPaymentsUseCase,
	public *usecases.PublicPaymentUseCase,
	webhook *usecases.WebhookUseCase,
) *Handler {
	return &Handler{
		integration:    integration,
		budgetPayments: budgetPayments,
		public:         public,
		webhook:        webhook,
	}
}

// SetupRoutes registers every payment route. Public and webhook routes are mounted
// WITHOUT auth; the webhook token and the budget share token are the only credentials.
func SetupRoutes(route *gin.RouterGroup, h *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	// Authenticated company integration settings.
	company := route.Group("/company/payments")
	{
		company.GET("/infinitepay", protectFactory(h.integration.Get, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		company.PUT("/infinitepay", protectFactory(h.integration.Upsert, roles.OwnerRole, roles.OrgAdminRole))
		company.DELETE("/infinitepay", protectFactory(h.integration.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}

	// Authenticated budget payment endpoints.
	budgets := route.Group("/budgets")
	{
		budgets.GET("/:id/payments", protectFactory(h.budgetPayments.List, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		budgets.POST("/:id/payments/link", protectFactory(h.budgetPayments.CreateLink, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}

	// Public (no-auth) pay/confirm endpoints.
	public := route.Group("/public/budgets")
	{
		public.POST("/:token/payments", h.public.Pay)
		public.POST("/:token/payments/confirm", h.public.Confirm)
	}

	// Webhook (no-auth; validated by the per-integration token in the path).
	webhooks := route.Group("/webhooks/infinitepay")
	{
		webhooks.POST("/:orgId/:token", h.webhook.Handle)
	}
}
