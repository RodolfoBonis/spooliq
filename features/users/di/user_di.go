package di

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	userRepositories "github.com/RodolfoBonis/spooliq/features/users/data/repositories"
	userServices "github.com/RodolfoBonis/spooliq/features/users/data/services"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	domainServices "github.com/RodolfoBonis/spooliq/features/users/domain/services"
	"github.com/RodolfoBonis/spooliq/features/users/presentation/handlers"
)

// Module provides the users feature module
var Module = fx.Module("users",
	fx.Provide(
		// Keycloak client
		func(keycloakConfig entities.KeyCloakDataEntity) *gocloak.GoCloak {
			return gocloak.NewClient(keycloakConfig.Host)
		},

		// Repositories
		fx.Annotate(
			userRepositories.NewKeycloakUserRepository,
			fx.As(new(repositories.UserRepository)),
		),

		// Services
		fx.Annotate(
			userServices.NewUserService,
			fx.As(new(domainServices.UserService)),
		),

		// Handlers
		handlers.NewUserHandler,

		// Keycloak configuration provider
		func() entities.KeyCloakDataEntity {
			return config.EnvKeyCloak()
		},
	),
	fx.Invoke(RegisterUserRoutes),
)

// RegisterUserRoutes registers the user routes
func RegisterUserRoutes(r *gin.Engine, handler *handlers.UserHandler) {
	v1 := r.Group("/v1")
	{
		// User management endpoints
		users := v1.Group("/users")
		{
			// Public endpoint for current user profile
			users.GET("/me", handler.GetCurrentUser)

			// Admin endpoints for user management
			users.GET("", handler.GetUsers)                          // List users (admin)
			users.POST("", handler.CreateUser)                       // Create user (admin)
			users.GET("/:id", handler.GetUserByID)                   // Get user by ID (admin or self)
			users.PATCH("/:id", handler.UpdateUser)                  // Update user (admin or self)
			users.DELETE("/:id", handler.DeleteUser)                 // Delete user (admin)
			users.PATCH("/:id/enabled", handler.SetUserEnabled)      // Enable/disable user (admin)
			users.PATCH("/:id/password", handler.ResetUserPassword)  // Reset password (admin)
			users.POST("/:id/roles", handler.AddUserRole)            // Add role (admin)
			users.DELETE("/:id/roles/:role", handler.RemoveUserRole) // Remove role (admin)
		}
	}
}
