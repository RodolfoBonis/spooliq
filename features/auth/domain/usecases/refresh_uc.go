package usecases

import (
	"net/http"
	"strings"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
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
		err := errors.NewAppError(coreEntities.ErrInvalidToken, "Token inválido", nil, nil)
		httpError := err.ToHTTPError()
		uc.Logger.LogError(ctx, "Refresh failed: missing token", err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		c.Abort()
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
		currentError := errors.UsecaseError(err.Error())
		httpError := currentError.ToHTTPError()
		uc.Logger.LogError(ctx, "Refresh falhou", currentError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		c.Abort()
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
