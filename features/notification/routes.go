package notification

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers the notification endpoints.
func Routes(router *gin.RouterGroup, service usecases.INotificationService, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	orgUsers := []string{roles.OwnerRole, roles.OrgAdminRole, roles.UserRole}
	notifications := router.Group("/notifications")
	notifications.GET("", protectFactory(service.List, orgUsers...))
	notifications.GET("/unread-count", protectFactory(service.UnreadCount, orgUsers...))
	notifications.POST("/read-all", protectFactory(service.MarkAllRead, orgUsers...))
	notifications.POST("/:id/read", protectFactory(service.MarkRead, orgUsers...))
}
