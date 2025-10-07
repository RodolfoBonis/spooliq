package brand

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all brand-related HTTP routes with authentication middleware.
func Routes(route *gin.RouterGroup, useCase usecases.IBrandUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	brands := route.Group("/brands")
	{
		brands.GET("/", protectFactory(useCase.FindAll, roles.AdminRole))
		brands.GET("/:id", protectFactory(useCase.FindByID, roles.AdminRole))
		brands.POST("/", protectFactory(useCase.Create, roles.AdminRole))
		brands.PUT("/:id", protectFactory(useCase.Update, roles.AdminRole))
	}
}
