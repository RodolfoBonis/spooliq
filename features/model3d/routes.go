package model3d

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	"github.com/gin-gonic/gin"
)

// setMultipartMemory is a middleware that overrides MaxMultipartMemory for upload routes,
// keeping the global default unchanged for other endpoints.
func setMultipartMemory(maxMemory int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMemory)
		c.Next()
	}
}

// Routes configures all model3d-related HTTP routes with authentication middleware.
func Routes(route *gin.RouterGroup, useCase usecases.IModel3DUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	models := route.Group("/models3d")
	{
		models.POST("", setMultipartMemory(64<<20), protectFactory(useCase.Upload, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("", protectFactory(useCase.FindAll, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("/:id", protectFactory(useCase.FindByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.GET("/by-customer/:customer_id", protectFactory(useCase.FindByCustomer, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		models.PUT("/:id", protectFactory(useCase.Update, roles.OwnerRole, roles.OrgAdminRole))
		models.DELETE("/:id", protectFactory(useCase.Delete, roles.OwnerRole, roles.OrgAdminRole))
	}
}
