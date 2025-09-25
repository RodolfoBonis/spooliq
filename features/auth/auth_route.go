package auth

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers authentication routes for the application.
func Routes(route *gin.RouterGroup, authUC usecases.AuthUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	route.POST("/login", authUC.ValidateLogin)
	route.POST("/logout", protectFactory(authUC.Logout, roles.UserRole))
	route.POST("/refresh", protectFactory(authUC.RefreshAuthToken, roles.UserRole))
	route.POST("/validate_token", protectFactory(authUC.ValidateToken, roles.UserRole))
}
