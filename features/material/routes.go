package material

import (
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures all material-related HTTP routes with authentication middleware and caching.
func Routes(route *gin.RouterGroup, useCase usecases.IMaterialUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, cacheMiddleware *middlewares.CacheMiddleware) {
	materials := route.Group("/materials")
	{
		materials.GET("/", protectFactory(useCase.FindAll, roles.AdminRole), cacheMiddleware.Cache15Min())
		materials.GET("/:id", protectFactory(useCase.FindByID, roles.AdminRole), cacheMiddleware.Cache1Hour())
		materials.POST("/", protectFactory(useCase.Create, roles.AdminRole))
		materials.PUT("/:id", protectFactory(useCase.Update, roles.AdminRole))
		materials.DELETE("/:id", protectFactory(useCase.Delete, roles.AdminRole))
	}
}
