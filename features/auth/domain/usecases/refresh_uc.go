package usecases

import (
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	"github.com/gin-gonic/gin"
)

// RefreshAuthToken renews the user's authentication tokens.
// @Summary Refresh Login Access Token
// @Schemes
// @Description Refresh the user's access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer refresh token"
// @Success 200 {object} entities.LoginResponseEntity "Tokens refreshed"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /refresh [post]
// @Example request {"Authorization": "Bearer <refresh-token>"}
// @Example response {"accessToken": "jwt-token", "refreshToken": "refresh-token", "expiresIn": 3600}
func (uc *authUseCaseImpl) RefreshAuthToken(c *gin.Context) {
	ctx := c.Request.Context()
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 1 {
		uc.Logger.Warning(ctx, "Refresh failed: missing token", logger.Fields{"ip": c.ClientIP()})
		errors.AbortWith(c, errors.Unauthorized("invalid_token", "Token inválido"))
		return
	}
	refreshToken := strings.Split(authHeader, " ")[1]
	token, err := uc.KeycloakClient.RefreshToken(
		ctx,
		refreshToken,
		uc.KeycloakAccessData.ClientID,
		uc.KeycloakAccessData.ClientSecret,
		uc.KeycloakAccessData.Realm,
	)
	if err != nil {
		uc.Logger.LogError(ctx, "Refresh falhou", err)
		// Only a rejected token is a 401 (the web logs the user out on 401). A
		// Keycloak outage or network error stays a retryable 500.
		var kcErr *gocloak.APIError
		if stderrors.As(err, &kcErr) && (kcErr.Code == http.StatusBadRequest || kcErr.Code == http.StatusUnauthorized) {
			errors.AbortWith(c, errors.Unauthorized("invalid_token", "Token inválido ou expirado"))
			return
		}
		errors.AbortWith(c, err)
		return
	}
	uc.Logger.Info(ctx, "Token refreshed successfully", logger.Fields{
		"ip": c.ClientIP(),
	})
	c.JSON(http.StatusOK, entities.LoginResponseEntity{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
	})
}
