package auth

import (
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"github.com/gin-gonic/gin"
)

// LoginHandler handles user login requests.
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
func LoginHandler(authUc usecases.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUc.ValidateLogin(c)
	}
}

// RegisterHandler handles user registration requests.
// @Summary User Registration
// @Schemes
// @Description Register a new user account in SpoolIQ
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.RequestRegisterEntity true "Registration data"
// @Success 201 {object} entities.RegisterResponseEntity "Successful registration"
// @Failure 400 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /auth/register [post]
// @Example request {"email": "user@example.com", "password": "SecurePass123", "firstName": "John", "lastName": "Doe"}
// @Example response {"message": "User registered successfully", "userID": "uuid", "email": "user@example.com"}
func RegisterHandler(authUc usecases.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUc.RegisterUser(c)
	}
}

// ForgotPasswordHandler handles password reset requests.
// @Summary Password Reset Request
// @Schemes
// @Description Request a password reset for a user account
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body entities.ForgotPasswordRequestEntity true "Password reset request data"
// @Success 200 {object} entities.ForgotPasswordResponseEntity "Password reset request sent"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /auth/forgot-password [post]
// @Example request {"email": "user@example.com"}
// @Example response {"message": "Password reset instructions have been sent to your email", "email": "user@example.com"}
func ForgotPasswordHandler(authUc usecases.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		authUc.ForgotPassword(c)
	}
}

// Routes registers authentication routes for the application.
func Routes(route *gin.RouterGroup, authUC usecases.AuthUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	route.POST("/auth/register", RegisterHandler(authUC))
	route.POST("/auth/forgot-password", ForgotPasswordHandler(authUC))
	route.POST("/login", LoginHandler(authUC))
	route.POST("/logout", protectFactory(authUC.Logout, "user"))
	route.POST("/refresh", protectFactory(authUC.RefreshAuthToken, "user"))
}
