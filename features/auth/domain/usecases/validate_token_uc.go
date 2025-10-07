package usecases

import (
	"net/http"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
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
// @Router /auth/validate [post]
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
		currentError := errors.NewAppError(entities.ErrUsecase, err.Error(), nil, err)
		httpError := currentError.ToHTTPError()
		uc.Logger.LogError(ctx, "Token validation failed", currentError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		c.Abort()
		return
	}
	isTokenValid := *rptResult.Active
	if !isTokenValid {
		currentError := errors.NewAppError(entities.ErrInvalidToken, "Token is invalid", nil, nil)
		httpError := currentError.ToHTTPError()
		uc.Logger.LogError(ctx, "Token is invalid", currentError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		c.Abort()
		return
	}
	uc.Logger.Info(ctx, "Token is valid", logger.Fields{
		"ip": c.ClientIP(),
	})
	c.JSON(http.StatusOK, true)
}
