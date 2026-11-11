package account

import (
	"net/http"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/account/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/account/domain/usecases"
	"github.com/gin-gonic/gin"
)

// Handler exposes the self-service account endpoints.
type Handler struct {
	uc *usecases.AccountUseCase
}

// NewHandler creates the account handler.
func NewHandler(uc *usecases.AccountUseCase) *Handler {
	return &Handler{uc: uc}
}

// SetupRoutes registers the account routes.
func SetupRoutes(route *gin.RouterGroup, h *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	route.POST("/password/forgot", h.ForgotPassword)

	anyUser := []string{roles.OwnerRole, roles.OrgAdminRole, roles.UserRole, roles.PlatformAdminRole}
	route.GET("/me", protectFactory(h.GetMe, anyUser...))
	route.PUT("/me", protectFactory(h.UpdateMe, anyUser...))
	route.POST("/me/password", protectFactory(h.ChangePassword, anyUser...))
}

// ForgotPassword sends a password reset e-mail.
// @Summary Forgot password
// @Description Sends a password reset link (via Keycloak) when the e-mail belongs to an active user. Always 204 for valid input, so it can't be used to discover accounts.
// @Tags account
// @Accept json
// @Param request body entities.ForgotPasswordRequest true "E-mail"
// @Success 204 "Accepted"
// @Failure 400 {object} errors.HTTPError
// @Failure 429 {object} errors.HTTPError
// @Router /password/forgot [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req entities.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.Respond(c, err)
		return
	}
	if err := h.uc.ForgotPassword(c.Request.Context(), &req, c.ClientIP()); err != nil {
		errors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// GetMe returns the authenticated user's profile.
// @Summary Get own profile
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.MeResponse
// @Failure 401 {object} errors.HTTPError
// @Router /me [get]
func (h *Handler) GetMe(c *gin.Context) {
	me, err := h.uc.GetMe(c.Request.Context(), claimsFrom(c))
	if err != nil {
		errors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, me)
}

// UpdateMe updates the authenticated user's own name.
// @Summary Update own profile
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body entities.UpdateMeRequest true "Profile"
// @Success 200 {object} entities.MeResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Router /me [put]
func (h *Handler) UpdateMe(c *gin.Context) {
	var req entities.UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.Respond(c, err)
		return
	}
	me, err := h.uc.UpdateMe(c.Request.Context(), claimsFrom(c), &req)
	if err != nil {
		errors.Respond(c, err)
		return
	}
	c.JSON(http.StatusOK, me)
}

// ChangePassword changes the authenticated user's own password.
// @Summary Change own password
// @Tags account
// @Accept json
// @Security BearerAuth
// @Param request body entities.ChangePasswordRequest true "Passwords"
// @Success 204 "Password changed"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Router /me/password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	var req entities.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.Respond(c, err)
		return
	}
	if err := h.uc.ChangePassword(c.Request.Context(), claimsFrom(c), &req); err != nil {
		errors.Respond(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func claimsFrom(c *gin.Context) usecases.Claims {
	claims := usecases.Claims{
		KeycloakID:     helpers.GetUserID(c),
		Email:          helpers.GetUserEmail(c),
		OrganizationID: helpers.GetOrganizationID(c),
	}
	if raw, ok := c.Get("claims"); ok {
		if jwt, ok := raw.(coreEntities.JWTClaim); ok {
			claims.Name = jwt.Name
		}
	}
	return claims
}
