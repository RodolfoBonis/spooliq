package usecases

import (
	"net/http"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
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
	// Create custom span for user login
	tracer := otel.Tracer("auth-service")
	ctx, span := logger.StartSpanWithLogger(c.Request.Context(), tracer, "auth.login", uc.Logger)
	var spanErr error
	defer func() {
		logger.EndSpanWithLogger(span, uc.Logger, spanErr)
	}()

	loginData := new(entities.RequestLoginEntity)
	// Log login attempt with trace context
	fields := logger.AddTraceToContext(ctx)
	fields["ip"] = c.ClientIP()
	fields["user_agent"] = c.Request.UserAgent()
	uc.Logger.Info(ctx, "Login attempt started", fields)

	err := c.BindJSON(&loginData)
	if err != nil {
		spanErr = err
		internalError := errors.NewAppError(coreEntities.ErrUsecase, err.Error(), nil, err)
		httpError := internalError.ToHTTPError()

		// Add trace context to error log
		errorFields := logger.AddTraceToContext(ctx)
		errorFields["error"] = err.Error()
		uc.Logger.Error(ctx, "Invalid login payload", errorFields)

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
		spanErr = err
		internalError := errors.NewAppError(coreEntities.ErrInvalidCredentials, "Invalid credentials", nil, err)
		httpError := internalError.ToHTTPError()

		// Add trace context to failed login log
		failFields := logger.AddTraceToContext(ctx)
		failFields["email"] = loginData.Email
		failFields["ip"] = c.ClientIP()
		failFields["error"] = err.Error()
		uc.Logger.Warning(ctx, "Login failed: invalid credentials", failFields)

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful login with trace context
	successFields := logger.AddTraceToContext(ctx)
	successFields["email"] = loginData.Email
	successFields["ip"] = c.ClientIP()
	successFields["user_agent"] = c.Request.UserAgent()
	uc.Logger.Info(ctx, "Login successful", successFields)
	c.JSON(http.StatusOK, entities.LoginResponseEntity{
		AccessToken:  jwt.AccessToken,
		RefreshToken: jwt.RefreshToken,
		ExpiresIn:    jwt.ExpiresIn,
	})
}
