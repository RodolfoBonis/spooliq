package usecases

import (
	"net/http"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	"github.com/gin-gonic/gin"
)

// ValidateLogin realiza a validação do login do usuário.
// Login validates user credentials and returns access and refresh tokens.
// @Summary User Login
// @Schemes
// @Description Authenticate user and return JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.RequestLoginEntity true "Login credentials"
// @Success 200 {object} entities.LoginResponseEntity "Successful login"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 403 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /auth/login [post]
// @Example request {"email": "user@example.com", "password": "string"}
// @Example response {"accessToken": "jwt-token", "refreshToken": "refresh-token", "expiresIn": 3600}
func (uc *authUseCaseImpl) ValidateLogin(c *gin.Context) {
	ctx := c.Request.Context()
	loginData := new(entities.RequestLoginEntity)
	
	// Log login attempt (automatic trace correlation via enhanced observability)
	uc.Logger.Info(ctx, "Login attempt started", map[string]interface{}{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	err := c.BindJSON(&loginData)
	if err != nil {
		internalError := errors.NewAppError(coreEntities.ErrUsecase, err.Error(), nil, err)
		httpError := internalError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.Logger.Error(ctx, "Invalid login payload", map[string]interface{}{
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}
	jwt, err := uc.KeycloakClient.Login(
		c,
		uc.KeycloakAccessData.ClientID,
		uc.KeycloakAccessData.ClientSecret,
		uc.KeycloakAccessData.Realm,
		loginData.Email,
		loginData.Password,
	)
	if err != nil {
		internalError := errors.NewAppError(coreEntities.ErrInvalidCredentials, "Invalid credentials", nil, err)
		httpError := internalError.ToHTTPError()

		// Enhanced logging with automatic trace correlation
		uc.Logger.Warning(ctx, "Login failed: invalid credentials", map[string]interface{}{
			"email": loginData.Email,
			"ip":    c.ClientIP(),
			"error": err.Error(),
		})

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful login with automatic trace correlation
	uc.Logger.Info(ctx, "Login successful", map[string]interface{}{
		"email":      loginData.Email,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})
	c.JSON(http.StatusOK, entities.LoginResponseEntity{
		AccessToken:  jwt.AccessToken,
		RefreshToken: jwt.RefreshToken,
		ExpiresIn:    jwt.ExpiresIn,
	})
}
