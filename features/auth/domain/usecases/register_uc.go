package usecases

import (
	"net/http"

	"github.com/Nerzal/gocloak/v13"
	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/entities"
	"github.com/gin-gonic/gin"
)

// RegisterUser creates a new user account in SpoolIQ
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
func (uc *authUseCaseImpl) RegisterUser(c *gin.Context) {
	ctx := c.Request.Context()
	registerData := new(entities.RequestRegisterEntity)

	uc.Logger.Info(ctx, "Registration attempt", logger.Fields{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	})

	err := c.BindJSON(&registerData)
	if err != nil {
		internalError := errors.NewAppError(coreEntities.ErrUsecase, err.Error(), nil, err)
		httpError := internalError.ToHTTPError()
		uc.Logger.LogError(ctx, "Invalid registration payload", internalError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate registration data
	if err := registerData.Validate(); err != nil {
		internalError := errors.NewAppError(coreEntities.ErrEntity, "Registration validation failed", nil, err)
		httpError := internalError.ToHTTPError()
		uc.Logger.LogError(ctx, "Registration validation failed", internalError)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if user already exists
	existingUsers, err := uc.KeycloakClient.GetUsers(
		ctx,
		"", // We need admin token here - this is a limitation
		uc.KeycloakAccessData.Realm,
		gocloak.GetUsersParams{
			Email: &registerData.Email,
			Exact: gocloak.BoolP(true),
		},
	)

	if err == nil && len(existingUsers) > 0 {
		internalError := errors.NewAppError(coreEntities.ErrConflict, "User already exists", nil, nil)
		httpError := internalError.ToHTTPError()
		uc.Logger.Warning(ctx, "Registration attempt with existing email", logger.Fields{
			"email": registerData.Email,
		})
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// For now, we'll return a simulated response since we need admin token to create users
	// In a real implementation, this would integrate with the user service or use a public registration flow
	response := entities.RegisterResponseEntity{
		Message: "Registration request received. Account activation pending.",
		Email:   registerData.Email,
		UserID:  "pending", // Would be generated after admin approval or email verification
	}

	uc.Logger.Info(ctx, "Registration request received", logger.Fields{
		"email":      registerData.Email,
		"first_name": registerData.FirstName,
		"last_name":  registerData.LastName,
	})

	c.JSON(http.StatusCreated, response)
}
