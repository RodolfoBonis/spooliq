package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/users/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/users/domain/services"
	"github.com/RodolfoBonis/spooliq/features/users/presentation/dto"
)

// UserHandler handles HTTP requests for user-related operations.
type UserHandler struct {
	userService services.UserService
	logger      logger.Logger
	validator   *validator.Validate
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userService services.UserService,
	logger logger.Logger,
) *UserHandler {
	if userService == nil {
		panic("userService cannot be nil")
	}
	if logger == nil {
		panic("logger cannot be nil")
	}
	return &UserHandler{
		userService: userService,
		logger:      logger,
		validator:   validator.New(),
	}
}

// GetUsers retrieves users with pagination and filtering (admin only)
// @Summary Get users list
// @Description Retrieves a paginated list of users with optional search filtering (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param search query string false "Search term for username, email, first name, or last name"
// @Param page query int false "Page number (default: 1)"
// @Param size query int false "Page size (default: 20, max: 100)"
// @Success 200 {object} dto.UsersListResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users [get]
// @Security BearerAuth
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Get query parameters
	var query dto.UserListQueryRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid query parameters", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind query parameters", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to domain query
	domainQuery := query.ToEntity()

	// Get users
	users, err := h.userService.GetUsers(c.Request.Context(), domainQuery, requesterID)
	if err != nil {
		h.handleError(c, err, "Failed to get users")
		return
	}

	// Convert to response
	response := dto.ToUsersListResponse(users, query.Page, query.Size)

	c.JSON(http.StatusOK, response)
}

// GetUserByID retrieves a user by ID (admin or self)
// @Summary Get user by ID
// @Description Retrieves a user by their ID (admin can get any user, users can get themselves)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if !h.validateUserID(c, userID) {
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get user
	user, err := h.userService.GetUserByID(c.Request.Context(), userID, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get user", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response
	response := dto.UserResponseFromEntity(user)

	c.JSON(http.StatusOK, response)
}

// GetCurrentUser retrieves the current user's profile
// @Summary Get current user profile
// @Description Retrieves the authenticated user's profile information
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserResponse
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/me [get]
// @Security BearerAuth
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get current user
	user, err := h.userService.GetCurrentUser(c.Request.Context(), requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get current user", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response
	response := dto.UserResponseFromEntity(user)

	c.JSON(http.StatusOK, response)
}

// CreateUser creates a new user (admin only)
// @Summary Create new user
// @Description Creates a new user account (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "User creation data"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users [post]
// @Security BearerAuth
func (h *UserHandler) CreateUser(c *gin.Context) {
	// Bind request
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind create user request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Create user request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to domain entity
	domainRequest := req.ToEntity()

	// Create user
	user, err := h.userService.CreateUser(c.Request.Context(), domainRequest, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to create user", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response
	response := dto.UserResponseFromEntity(user)

	c.JSON(http.StatusCreated, response)
}

// UpdateUser updates a user (admin or self with restrictions)
// @Summary Update user
// @Description Updates user information (admin can update any user, users can update themselves with restrictions)
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/{id} [patch]
// @Security BearerAuth
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Bind request
	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind update user request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Update user request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to domain entity
	domainRequest := req.ToEntity()

	// Update user
	user, err := h.userService.UpdateUser(c.Request.Context(), userID, domainRequest, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to update user", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response
	response := dto.UserResponseFromEntity(user)

	c.JSON(http.StatusOK, response)
}

// DeleteUser deletes a user (admin only)
// @Summary Delete user
// @Description Deletes a user account (admin only)
// @Tags users
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
// @Security BearerAuth
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Delete user
	err := h.userService.DeleteUser(c.Request.Context(), userID, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to delete user", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// SetUserEnabled enables or disables a user account (admin only)
// @Summary Enable/disable user
// @Description Enables or disables a user account (admin only)
// @Tags users
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
// @Security BearerAuth
func (h *UserHandler) SetUserEnabled(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Bind request
	var req dto.SetUserEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind set enabled request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Set user enabled status
	err := h.userService.SetUserEnabled(c.Request.Context(), userID, req.Enabled, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to set user enabled status", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// ResetUserPassword resets a user's password (admin only)
// @Summary Reset user password
// @Description Resets a user's password (admin only)
// @Tags users
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
// @Security BearerAuth
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Bind request
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind reset password request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Reset password request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Reset password
	err := h.userService.ResetUserPassword(c.Request.Context(), userID, req.NewPassword, req.Temporary, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to reset user password", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// AddUserRole adds a role to a user (admin only)
// @Summary Add role to user
// @Description Adds a role to a user (admin only)
// @Tags users
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
// @Security BearerAuth
func (h *UserHandler) AddUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Bind request
	var req dto.UserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid request format", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind add role request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Validation failed", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Add role request validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Add role
	err := h.userService.AddUserRole(c.Request.Context(), userID, req.Role, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to add user role", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveUserRole removes a role from a user (admin only)
// @Summary Remove role from user
// @Description Removes a role from a user (admin only)
// @Tags users
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
// @Security BearerAuth
func (h *UserHandler) RemoveUserRole(c *gin.Context) {
	userID := c.Param("id")
	role := c.Param("role")

	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	if role == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Role is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Role is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Remove role
	err := h.userService.RemoveUserRole(c.Request.Context(), userID, role, requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to remove user role", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetUserStats retrieves user statistics (admin only)
// @Summary Get user statistics
// @Description Retrieves user statistics including total, active, inactive, suspended, and admin counts (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserStatsResponse
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /users/stats [get]
// @Security BearerAuth
func (h *UserHandler) GetUserStats(c *gin.Context) {
	// Get requester ID
	requesterID := c.GetString("user_id")
	if requesterID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrUnauthorized, "User not authenticated", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get user stats
	stats, err := h.userService.GetUserStats(c.Request.Context(), requesterID)
	if err != nil {
		appError := h.mapDomainError(err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to get user stats", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response
	response := dto.UserStatsResponse{
		Total:     stats.Total,
		Active:    stats.Active,
		Inactive:  stats.Inactive,
		Suspended: stats.Suspended,
		Admins:    stats.Admins,
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *UserHandler) handleError(c *gin.Context, err error, message string) {
	appError := h.mapDomainError(err)
	if appError == nil {
		// Fallback for nil appError
		appError = coreErrors.NewAppError(coreEntities.ErrService, "Internal server error", nil, err)
	}
	httpError := appError.ToHTTPError()
	if httpError == nil {
		// Fallback for nil httpError
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if h.logger != nil {
		h.logger.LogError(c.Request.Context(), message, appError)
	}
	c.JSON(httpError.StatusCode, httpError)
}

func (h *UserHandler) mapDomainError(err error) *coreErrors.AppError {
	if err == nil {
		return coreErrors.NewAppError(coreEntities.ErrService, "Unknown error", nil, nil)
	}

	// Check for specific domain errors using errors.Is to handle wrapped errors
	if errors.Is(err, entities.ErrUserNotFound) {
		return coreErrors.NewAppError(coreEntities.ErrNotFound, "User not found", nil, err)
	}
	if errors.Is(err, entities.ErrUserAlreadyExists) {
		return coreErrors.NewAppError(coreEntities.ErrConflict, "User already exists", nil, err)
	}
	if errors.Is(err, entities.ErrUnauthorized) {
		return coreErrors.NewAppError(coreEntities.ErrUnauthorized, "Unauthorized", nil, err)
	}
	if errors.Is(err, entities.ErrInvalidUserData) {
		return coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid user data", nil, err)
	}
	if errors.Is(err, entities.ErrPasswordTooWeak) {
		return coreErrors.NewAppError(coreEntities.ErrEntity, "Password does not meet security requirements", nil, err)
	}

	// Default case for unknown errors
	return coreErrors.NewAppError(coreEntities.ErrService, "Internal server error", nil, err)
}

// validateUserID validates that the given ID is a valid UUID format
func (h *UserHandler) validateUserID(c *gin.Context, userID string) bool {
	if userID == "" {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "User ID is required", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User ID is required", appError)
		c.JSON(httpError.StatusCode, httpError)
		return false
	}

	if _, err := uuid.Parse(userID); err != nil {
		appError := coreErrors.NewAppError(coreEntities.ErrEntity, "Invalid user ID format - must be a valid UUID", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Invalid user ID format", appError)
		c.JSON(httpError.StatusCode, httpError)
		return false
	}

	return true
}
