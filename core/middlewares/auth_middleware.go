package middlewares

import (
	"encoding/json"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"

	jsonToken "github.com/golang-jwt/jwt/v4"
)

// NewProtectMiddleware creates a new authentication middleware.
func NewProtectMiddleware(logger logger.Logger, authService *services.AuthService) func(handler gin.HandlerFunc, role string) gin.HandlerFunc {
	return func(handler gin.HandlerFunc, role string) gin.HandlerFunc {
		return func(c *gin.Context) {
			ctx := c.Request.Context()
			requestID, _ := c.Get("requestID")
			keycloakDataAccess := config.EnvKeyCloak()
			authHeader := c.GetHeader("Authorization")

			if len(authHeader) < 1 {
				err := errors.NewAppError(entities.ErrInvalidToken, "Token ausente", nil, nil)
				httpError := err.ToHTTPError()
				logger.LogError(ctx, "Auth failed: missing token", err)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			accessToken := strings.Split(authHeader, " ")[1]

			rptResult, err := authService.GetClient().RetrospectToken(
				c,
				accessToken,
				keycloakDataAccess.ClientID,
				keycloakDataAccess.ClientSecret,
				keycloakDataAccess.Realm,
			)

			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				httpError := appError.ToHTTPError()
				logger.LogError(ctx, "Auth failed: token introspection error", appError)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			isTokenValid := *rptResult.Active

			if !isTokenValid {
				err := errors.NewAppError(entities.ErrInvalidToken, "Token inválido", nil, nil)
				httpError := err.ToHTTPError()
				logger.LogError(ctx, "Auth failed: token invalid", err)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			token, _, err := authService.GetClient().DecodeAccessToken(
				c,
				accessToken,
				keycloakDataAccess.Realm,
			)

			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				httpError := appError.ToHTTPError()
				logger.LogError(ctx, "Auth failed: decode token error", appError)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			claims := token.Claims.(jsonToken.MapClaims)

			jsonData, _ := json.Marshal(claims)

			var userClaim entities.JWTClaim

			err = json.Unmarshal(jsonData, &userClaim)
			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				httpError := appError.ToHTTPError()
				logger.LogError(ctx, "Auth failed: unmarshal claims error", appError)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			keyCloakData := config.EnvKeyCloak()
			client := userClaim.ResourceAccess[keyCloakData.ClientID].(map[string]interface{})
			rolesBytes, _ := json.Marshal(client["roles"])
			err = json.Unmarshal(rolesBytes, &userClaim.Roles)
			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				httpError := appError.ToHTTPError()
				logger.LogError(ctx, "Auth failed: unmarshal roles error", appError)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			containsRole := userClaim.Roles.Contains(role)

			if !containsRole {
				appError := errors.NewAppError(entities.ErrUnauthorized, "Perfil de acesso necessário ausente", nil, nil)
				httpError := appError.ToHTTPError()
				logger.LogError(ctx, "Auth failed: missing required role", appError)
				c.AbortWithStatusJSON(httpError.StatusCode, httpError)
				c.Abort()
				return
			}

			logger.Info(ctx, "Auth success", map[string]interface{}{
				"request_id": requestID,
				"ip":         c.ClientIP(),
				"role":       role,
				"user_roles": userClaim.Roles,
				"user_id":    userClaim.ID,
				"email":      userClaim.Email,
			})

			// Set claims and individual user data for easy access
			c.Set("claims", userClaim)
			c.Set("user_id", userClaim.ID.String())
			c.Set("user_email", userClaim.Email)
			c.Set("user_role", role)
			c.Set("user_roles", userClaim.Roles)

			handler(c)
		}
	}
}

// NewOptionalAuthMiddleware creates a middleware that extracts user information if a valid token is present,
// but doesn't fail if no token is provided (for public endpoints that behave differently with auth)
func NewOptionalAuthMiddleware(logger logger.Logger, authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		keycloakDataAccess := config.EnvKeyCloak()
		authHeader := c.GetHeader("Authorization")

		// If no Authorization header, continue without authentication
		if len(authHeader) < 1 {
			c.Next()
			return
		}

		// Extract token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}
		accessToken := tokenParts[1]

		// Try to validate token
		rptResult, err := authService.GetClient().RetrospectToken(
			c,
			accessToken,
			keycloakDataAccess.ClientID,
			keycloakDataAccess.ClientSecret,
			keycloakDataAccess.Realm,
		)

		if err != nil {
			// If token validation fails, continue without authentication
			c.Next()
			return
		}

		isTokenValid := *rptResult.Active

		if !isTokenValid {
			// If token is invalid, continue without authentication
			c.Next()
			return
		}

		// Try to decode token
		token, _, err := authService.GetClient().DecodeAccessToken(
			c,
			accessToken,
			keycloakDataAccess.Realm,
		)

		if err != nil {
			// If token decode fails, continue without authentication
			c.Next()
			return
		}

		// Extract claims
		claims := token.Claims.(jsonToken.MapClaims)
		jsonData, _ := json.Marshal(claims)
		var userClaim entities.JWTClaim

		err = json.Unmarshal(jsonData, &userClaim)
		if err != nil {
			// If claims extraction fails, continue without authentication
			c.Next()
			return
		}

		// Extract roles
		keyCloakData := config.EnvKeyCloak()
		client := userClaim.ResourceAccess[keyCloakData.ClientID].(map[string]interface{})
		rolesBytes, _ := json.Marshal(client["roles"])
		err = json.Unmarshal(rolesBytes, &userClaim.Roles)
		if err != nil {
			// If roles extraction fails, continue without authentication
			c.Next()
			return
		}

		logger.Info(ctx, "Optional auth success", map[string]interface{}{
			"ip":         c.ClientIP(),
			"user_id":    userClaim.ID,
			"email":      userClaim.Email,
			"user_roles": userClaim.Roles,
		})

		// Set user information for use in handlers
		c.Set("claims", userClaim)
		c.Set("user_id", userClaim.ID.String())
		c.Set("user_email", userClaim.Email)
		c.Set("user_roles", userClaim.Roles)

		c.Next()
	}
}
