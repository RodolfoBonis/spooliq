package users

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	createUserUC *usecases.CreateUserUseCase
	listUsersUC  *usecases.ListUsersUseCase
	findUserUC   *usecases.FindUserUseCase
	updateUserUC *usecases.UpdateUserUseCase
	deleteUserUC *usecases.DeleteUserUseCase
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	createUserUC *usecases.CreateUserUseCase,
	listUsersUC *usecases.ListUsersUseCase,
	findUserUC *usecases.FindUserUseCase,
	updateUserUC *usecases.UpdateUserUseCase,
	deleteUserUC *usecases.DeleteUserUseCase,
) *Handler {
	return &Handler{
		createUserUC: createUserUC,
		listUsersUC:  listUsersUC,
		findUserUC:   findUserUC,
		updateUserUC: updateUserUC,
		deleteUserUC: deleteUserUC,
	}
}

// SetupRoutes configures user-related HTTP routes
func SetupRoutes(route *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	users := route.Group("/users")
	{
		// All users can view their own info; Owner and OrgAdmin can view all users
		users.GET("", protectFactory(handler.ListUsers, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		users.GET("/:id", protectFactory(handler.GetUser, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Only Owner and OrgAdmin can create users
		users.POST("", protectFactory(handler.CreateUser, roles.OwnerRole, roles.OrgAdminRole))

		// Owner and OrgAdmin can update users (with permission checks in use case)
		users.PUT("/:id", protectFactory(handler.UpdateUser, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Only Owner and OrgAdmin can delete users (with permission checks in use case)
		users.DELETE("/:id", protectFactory(handler.DeleteUser, roles.OwnerRole, roles.OrgAdminRole))
	}
}

// CreateUser handles user creation
// @Summary Create a new user
// @Description Creates a new user within the organization (Owner and OrgAdmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body entities.CreateUserRequest true "User creation request"
// @Success 201 {object} entities.UserEntity "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 409 {object} map[string]string "User already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Get organization ID from context
	organizationIDStr := helpers.GetOrganizationIDString(c)
	if organizationIDStr == "" {
		appError := errors.UnauthorizedError("Organization ID not found")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Parse request body
	var req entities.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.BadRequestError("Invalid request body")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Execute use case
	user, err := h.createUserUC.Execute(ctx, organizationIDStr, userRoles, &req)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to create user")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// ListUsers handles listing users
// @Summary List all users
// @Description Lists all users within the organization (Owner and OrgAdmin only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} entities.UserEntity "List of users"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	// Get organization ID from context
	organizationIDStr := helpers.GetOrganizationIDString(c)
	if organizationIDStr == "" {
		appError := errors.UnauthorizedError("Organization ID not found")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get user roles from context
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	users, err := h.listUsersUC.Execute(ctx, organizationIDStr, userRoles)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to list users")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetUser handles getting a user by ID
// @Summary Get user by ID
// @Description Gets a user by ID (Owner, OrgAdmin, or self)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} entities.UserEntity "User details"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid user ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get organization ID from context
	organizationIDStr := helpers.GetOrganizationIDString(c)
	if organizationIDStr == "" {
		appError := errors.UnauthorizedError("Organization ID not found")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get current user ID and roles from context
	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	user, err := h.findUserUC.Execute(ctx, userID, organizationIDStr, currentUserID, userRoles)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to get user")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles updating a user
// @Summary Update user
// @Description Updates a user (Owner can update anyone, OrgAdmin can update users only, not self)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Param request body entities.UpdateUserRequest true "User update request"
// @Success 200 {object} entities.UserEntity "User updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid user ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get organization ID from context
	organizationIDStr := helpers.GetOrganizationIDString(c)
	if organizationIDStr == "" {
		appError := errors.UnauthorizedError("Organization ID not found")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get current user ID and roles from context
	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	// Parse request body
	var req entities.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.BadRequestError("Invalid request body")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Execute use case
	user, err := h.updateUserUC.Execute(ctx, userID, organizationIDStr, currentUserID, userRoles, &req)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to update user")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser handles deleting a user
// @Summary Delete user
// @Description Deletes a user (Owner can delete anyone except self, OrgAdmin can delete users only)
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 204 "User deleted successfully"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 403 {object} map[string]string "Forbidden"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse user ID from URL
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		appError := errors.BadRequestError("Invalid user ID")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get organization ID from context
	organizationIDStr := helpers.GetOrganizationIDString(c)
	if organizationIDStr == "" {
		appError := errors.UnauthorizedError("Organization ID not found")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get current user ID and roles from context
	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	// Execute use case
	err = h.deleteUserUC.Execute(ctx, userID, organizationIDStr, currentUserID, userRoles)
	if err != nil {
		if appError, ok := err.(*errors.AppError); ok {
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
		appError := errors.InternalServerError("Failed to delete user")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	c.Status(http.StatusNoContent)
}
