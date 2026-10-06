package model3d

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all model3d-related HTTP routes with authentication middleware.
// Note: MaxMultipartMemory stays at the global default (32MB). Uploads larger than that
// are automatically spilled to temp files by Gin, which is safe. The actual file size
// limit (50MB) is enforced in the Upload use case via fileHeader.Size validation.
func Routes(route *gin.RouterGroup, useCase usecases.IModel3DUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	models := route.Group("/models3d")
	{
		models.POST("", protectFactory(useCase.Upload, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("/by-customer/:customer_id", protectFactory(useCase.FindByCustomer, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("/:id/file", protectFactory(useCase.StreamFile, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
		models.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
