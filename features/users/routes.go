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

// userListOptions is the single source of truth for how GET /users interprets
// pagination, search and sort. Free-text search (q) matches name or email;
// sorting is restricted to name/email/created_at.
func userListOptions() helpers.ListQueryOptions {
	return helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist: map[string]string{
			"name":       "name",
			"email":      "email",
			"created_at": "created_at",
		},
		DefaultSort: "created_at",
	}
}

// requireOrganizationID extracts the organization_id from context, writing a 401
// and returning ok=false when it is missing.
func requireOrganizationID(c *gin.Context) (string, bool) {
	organizationID := helpers.GetOrganizationIDString(c)
	if organizationID == "" {
		errors.Respond(c, errors.Unauthorized("organization_id_missing", "Organização não encontrada no contexto"))
		return "", false
	}
	return organizationID, true
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
// @Failure 400 {object} errors.HTTPError "Invalid request"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 409 {object} errors.HTTPError "User already exists"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	userRoles := helpers.GetUserRoles(c)

	var req entities.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.Respond(c, err)
		return
	}

	user, err := h.createUserUC.Execute(ctx, organizationID, userRoles, &req)
	if err != nil {
		errors.Respond(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// ListUsers handles listing users, paginated with the standard envelope.
// @Summary List users
// @Description Lists users within the organization (Owner and OrgAdmin only), paginated with the standard envelope.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 20, max 100)"
// @Param q query string false "Free-text search on name or email"
// @Param sort_by query string false "Sort field: name, email or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[entities.UserEntity] "Paginated users"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	userRoles := helpers.GetUserRoles(c)

	q := helpers.ParseListQuery(c, userListOptions())
	users, total, err := h.listUsersUC.Execute(ctx, organizationID, userRoles, q)
	if err != nil {
		errors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(users, total, q))
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
// @Failure 400 {object} errors.HTTPError "Invalid user ID"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 404 {object} errors.HTTPError "User not found"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.Respond(c, errors.BadRequest("invalid_user_id", "ID de usuário inválido"))
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	user, err := h.findUserUC.Execute(ctx, userID, organizationID, currentUserID, userRoles)
	if err != nil {
		errors.Respond(c, err)
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
// @Failure 400 {object} errors.HTTPError "Invalid request"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 404 {object} errors.HTTPError "User not found"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.Respond(c, errors.BadRequest("invalid_user_id", "ID de usuário inválido"))
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	var req entities.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.Respond(c, err)
		return
	}

	user, err := h.updateUserUC.Execute(ctx, userID, organizationID, currentUserID, userRoles, &req)
	if err != nil {
		errors.Respond(c, err)
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
// @Failure 400 {object} errors.HTTPError "Invalid user ID"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Failure 403 {object} errors.HTTPError "Forbidden"
// @Failure 404 {object} errors.HTTPError "User not found"
// @Failure 500 {object} errors.HTTPError "Internal server error"
// @Router /users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	ctx := c.Request.Context()

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.Respond(c, errors.BadRequest("invalid_user_id", "ID de usuário inválido"))
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	currentUserID := helpers.GetUserID(c)
	userRoles := helpers.GetUserRoles(c)

	if err := h.deleteUserUC.Execute(ctx, userID, organizationID, currentUserID, userRoles); err != nil {
		errors.Respond(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
