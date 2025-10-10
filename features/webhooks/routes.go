package webhooks

import (
	"github.com/RodolfoBonis/spooliq/features/webhooks/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for webhook operations
type Handler struct {
	asaasWebhookUC *usecases.AsaasWebhookUseCase
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(asaasWebhookUC *usecases.AsaasWebhookUseCase) *Handler {
	return &Handler{
		asaasWebhookUC: asaasWebhookUC,
	}
}

// SetupRoutes configures webhook routes (public, no authentication)
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	webhooks := router.Group("/webhooks")
	{
		// Asaas webhook endpoint (public, validated by signature)
		webhooks.POST("/asaas", handler.asaasWebhookUC.HandleWebhook)
	}
}
