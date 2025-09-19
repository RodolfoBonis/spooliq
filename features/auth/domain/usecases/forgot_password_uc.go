package usecases

import (
	"net/http"

	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	"github.com/gin-gonic/gin"
)

// ForgotPassword handles password reset requests in SpoolIQ
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
func (uc *authUseCaseImpl) ForgotPassword(c *gin.Context) {
	ctx := c.Request.Context()
	forgotPasswordData := new(entities.ForgotPasswordRequestEntity)

	uc.Logger.Info(ctx, "Password reset request", logger.Fields{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	err := c.BindJSON(&forgotPasswordData)
	if err != nil {
		internalError := errors.NewAppError(coreEntities.ErrUsecase, err.Error(), nil, err)
		httpError := internalError.ToHTTPError()
		uc.Logger.LogError(ctx, "Invalid forgot password payload", internalError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate forgot password data
	if err := forgotPasswordData.Validate(); err != nil {
		internalError := errors.NewAppError(coreEntities.ErrEntity, "Forgot password validation failed", nil, err)
		httpError := internalError.ToHTTPError()
		uc.Logger.LogError(ctx, "Forgot password validation failed", internalError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// In a real implementation, this would:
	// 1. Check if user exists in Keycloak
	// 2. Generate a password reset token
	// 3. Send email with reset instructions
	// 4. Store the reset token temporarily

	// For now, we'll return a simulated response
	response := entities.ForgotPasswordResponseEntity{
		Message: "If an account with that email exists, password reset instructions have been sent.",
		Email:   forgotPasswordData.Email,
	}

	uc.Logger.Info(ctx, "Password reset request processed", logger.Fields{
		"email": forgotPasswordData.Email,
	})

	c.JSON(http.StatusOK, response)
}
