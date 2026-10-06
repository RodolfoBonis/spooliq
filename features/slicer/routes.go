// Package slicer wires the slicer analysis feature's HTTP routes.
package slicer

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes configures the slicer routes under the authenticated group.
func Routes(route *gin.RouterGroup, useCase usecases.ISlicerUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	group := route.Group("/slicer")
	{
		group.POST("/analyze", protectFactory(useCase.Analyze, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	}
}
