package brand

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/gin-gonic/gin"
)

func Routes(route *gin.RouterGroup, useCase usecases.IBrandUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	brands := route.Group("/brands")
	{
		brands.POST("/", protectFactory(useCase.Create, roles.AdminRole))
	}
}
