package uploads

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all upload routes
func Routes(route *gin.RouterGroup, useCase usecases.IUploadUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	uploadRoutes := route.Group("/uploads")
	{
		// All users can upload files
		uploadRoutes.POST("/logo", protectFactory(useCase.UploadLogo, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		uploadRoutes.POST("/file", protectFactory(useCase.UploadFile, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}
}
