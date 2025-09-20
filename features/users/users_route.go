package users

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/services"
	"github.com/RodolfoBonis/spooliq/features/users/presentation/handlers"
	"github.com/gin-gonic/gin"
)

// GetCurrentUserHandler handles getting current user profile.
// @Summary Get current user profile
// @Schemes
// @Description Retrieves the authenticated user's profile information
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserResponse "Successfully retrieved current user"
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/me [get]
// @Security Bearer
func GetCurrentUserHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.GetCurrentUser
}

// GetUsersHandler handles getting users list.
// @Summary Get users list
// @Schemes
// @Description Retrieves a paginated list of users with optional search filtering (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param search query string false "Search term for username, email, first name, or last name"
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.UsersListResponse "Successfully retrieved users"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users [get]
// @Security Bearer
func GetUsersHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.GetUsers
}

// CreateUserHandler handles creating a new user.
// @Summary Create new user
// @Schemes
// @Description Creates a new user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "User creation data"
// @Success 201 {object} dto.UserResponse "Successfully created user"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users [post]
// @Security Bearer
func CreateUserHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.CreateUser
}

// GetUserByIDHandler handles getting a user by ID.
// @Summary Get user by ID
// @Schemes
// @Description Retrieves a user by their ID (admin can get any user, users can get themselves)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserResponse "Successfully retrieved user"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id} [get]
// @Security Bearer
func GetUserByIDHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.GetUserByID
}

// UpdateUserHandler handles updating a user.
// @Summary Update user
// @Schemes
// @Description Updates user information (admin can update any user, users can update themselves with restrictions)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.UserResponse "Successfully updated user"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id} [patch]
// @Security Bearer
func UpdateUserHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.UpdateUser
}

// DeleteUserHandler handles deleting a user.
// @Summary Delete user
// @Schemes
// @Description Deletes a user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id} [delete]
// @Security Bearer
func DeleteUserHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.DeleteUser
}

// SetUserEnabledHandler handles enabling/disabling a user.
// @Summary Enable/disable user
// @Schemes
// @Description Enables or disables a user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.SetUserEnabledRequest true "Enable/disable data"
// @Success 204 "User status updated successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id}/enabled [patch]
// @Security Bearer
func SetUserEnabledHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.SetUserEnabled
}

// ResetUserPasswordHandler handles resetting a user's password.
// @Summary Reset user password
// @Schemes
// @Description Resets a user's password (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.ResetPasswordRequest true "Password reset data"
// @Success 204 "Password reset successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id}/password [patch]
// @Security Bearer
func ResetUserPasswordHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.ResetUserPassword
}

// AddUserRoleHandler handles adding a role to a user.
// @Summary Add role to user
// @Schemes
// @Description Adds a role to a user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.UserRoleRequest true "Role data"
// @Success 204 "Role added successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id}/roles [post]
// @Security Bearer
func AddUserRoleHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.AddUserRole
}

// RemoveUserRoleHandler handles removing a role from a user.
// @Summary Remove role from user
// @Schemes
// @Description Removes a role from a user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param role path string true "Role name"
// @Success 204 "Role removed successfully"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id}/roles/{role} [delete]
// @Security Bearer
func RemoveUserRoleHandler(userService services.UserService, logger logger.Logger) gin.HandlerFunc {
	handler := handlers.NewUserHandler(userService, logger)
	return handler.RemoveUserRole
}

// Routes registers user routes for the application.
func Routes(route *gin.RouterGroup, userService services.UserService, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc, logger logger.Logger) {
	users := route.Group("/users")

	// User self-management routes
	users.GET("/me", protectFactory(GetCurrentUserHandler(userService, logger), roles.UserRole))

	// Admin-only routes
	users.GET("", protectFactory(GetUsersHandler(userService, logger), roles.AdminRole))
	users.POST("", protectFactory(CreateUserHandler(userService, logger), roles.AdminRole))
	users.GET("/:id", protectFactory(GetUserByIDHandler(userService, logger), roles.UserRole))
	users.PATCH("/:id", protectFactory(UpdateUserHandler(userService, logger), roles.UserRole))
	users.DELETE("/:id", protectFactory(DeleteUserHandler(userService, logger), roles.AdminRole))
	users.PATCH("/:id/enabled", protectFactory(SetUserEnabledHandler(userService, logger), roles.AdminRole))
	users.PATCH("/:id/password", protectFactory(ResetUserPasswordHandler(userService, logger), roles.AdminRole))
	users.POST("/:id/roles", protectFactory(AddUserRoleHandler(userService, logger), roles.AdminRole))
	users.DELETE("/:id/roles/:role", protectFactory(RemoveUserRoleHandler(userService, logger), roles.AdminRole))
}
