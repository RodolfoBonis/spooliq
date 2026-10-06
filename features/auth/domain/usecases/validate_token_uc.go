package usecases

import (
	"net/http"
	"strings"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/gin-gonic/gin"
)

// ValidateToken checks if the provided access token is valid.
// @Summary Validate Auth Token
// @Schemes
// @Description Validate the current access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Success 200 {object} bool "Token is valid"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /validate_token [post]
// @Example request {"Authorization": "Bearer <access-token>"}
// @Example response true
func (uc *authUseCaseImpl) ValidateToken(c *gin.Context) {
	ctx := c.Request.Context()
	authorization := c.GetHeader("Authorization")
	uc.Logger.Info(ctx, "Token validation attempt", logger.Fields{
		"ip": c.ClientIP(),
	})
	token := strings.Split(authorization, " ")[1]
	rptResult, err := uc.KeycloakClient.RetrospectToken(
		ctx,
		token,
		uc.KeycloakAccessData.ClientID,
		uc.KeycloakAccessData.ClientSecret,
		uc.KeycloakAccessData.Realm,
	)
	if err != nil {
		uc.Logger.LogError(ctx, "Token validation failed", err)
		errors.AbortWith(c, errors.Unauthorized("invalid_token", "Token inválido"))
		return
	}
	isTokenValid := *rptResult.Active
	if !isTokenValid {
		uc.Logger.Warning(ctx, "Token is invalid", logger.Fields{"ip": c.ClientIP()})
		errors.AbortWith(c, errors.Unauthorized("invalid_token", "Token inválido"))
		return
	}
	uc.Logger.Info(ctx, "Token is valid", logger.Fields{
		"ip": c.ClientIP(),
	})
	c.JSON(http.StatusOK, true)
}
