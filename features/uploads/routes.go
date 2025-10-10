package uploads

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers all upload routes
func Routes(route *gin.RouterGroup, useCase usecases.IUploadUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	uploadRoutes := route.Group("/uploads")
	{
		// All upload routes require UserRole
		uploadRoutes.POST("/logo", protectFactory(useCase.UploadLogo, roles.UserRole))
		uploadRoutes.POST("/file", protectFactory(useCase.UploadFile, roles.UserRole))
	}
}
