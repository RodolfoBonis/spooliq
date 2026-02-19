package activity

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all activity-related HTTP routes with authentication middleware.
func Routes(router *gin.RouterGroup, activityService usecases.IActivityService, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	activities := router.Group("/activities")
	{
		activities.GET("", protectFactory(activityService.ListActivities, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}
}
