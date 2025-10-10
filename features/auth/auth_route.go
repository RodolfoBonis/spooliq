package auth

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers authentication routes for the application.
func Routes(route *gin.RouterGroup, authUC usecases.AuthUseCase, registerUC *usecases.RegisterUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	// Public routes
	route.POST("/register", registerUC.Register)
	route.POST("/login", authUC.ValidateLogin)
	
	// Protected routes
	route.POST("/logout", protectFactory(authUC.Logout, roles.UserRole))
	route.POST("/refresh", protectFactory(authUC.RefreshAuthToken, roles.UserRole))
	route.POST("/validate_token", protectFactory(authUC.ValidateToken, roles.UserRole))
}
