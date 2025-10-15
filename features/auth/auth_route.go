package auth

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Routes registers authentication routes for the application.
func Routes(route *gin.RouterGroup, authUC usecases.AuthUseCase, registerUC *usecases.RegisterUseCase, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	// Public routes
	route.POST("/register", registerUC.Register)
	route.POST("/login", authUC.ValidateLogin)

	// Protected routes - all authenticated users can access
	route.POST("/logout", protectFactory(authUC.Logout, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	route.POST("/refresh", protectFactory(authUC.RefreshAuthToken, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
	route.POST("/validate_token", protectFactory(authUC.ValidateToken, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
}
