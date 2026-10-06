package middlewares

import (
	"encoding/json"
	"strings"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/core/types"
	"github.com/gin-gonic/gin"

	jsonToken "github.com/golang-jwt/jwt/v4"
)

// NewProtectMiddleware creates a new authentication middleware with subscription check.
// Accepts multiple roles - user must have at least one of the specified roles to access the endpoint.
func NewProtectMiddleware(logger logger.Logger, authService *services.AuthService, subscriptionMiddleware *SubscriptionMiddleware) func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc {
	return func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc {
		return func(c *gin.Context) {
			ctx := c.Request.Context()
			requestID, _ := c.Get("requestID")
			keycloakDataAccess := config.EnvKeyCloak()
			authHeader := c.GetHeader("Authorization")

			if len(authHeader) < 1 {
				appErr := errors.NewAppError(entities.ErrInvalidToken, "Token ausente", nil, nil)
				logger.LogError(ctx, "Auth failed: missing token", appErr)
				errors.AbortWith(c, appErr)
				return
			}

			// Verificar se o header contém "Bearer " e extrair o token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				appErr := errors.NewAppError(entities.ErrInvalidToken, "Formato de token inválido", nil, nil)
				logger.LogError(ctx, "Auth failed: invalid token format", appErr)
				errors.AbortWith(c, appErr)
				return
			}

			accessToken := parts[1]

			rptResult, err := authService.GetClient().RetrospectToken(
				c,
				accessToken,
				keycloakDataAccess.ClientID,
				keycloakDataAccess.ClientSecret,
				keycloakDataAccess.Realm,
			)

			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				logger.LogError(ctx, "Auth failed: token introspection error", appError)
				errors.AbortWith(c, appError)
				return
			}

			isTokenValid := *rptResult.Active

			if !isTokenValid {
				appErr := errors.NewAppError(entities.ErrInvalidToken, "Token inválido", nil, nil)
				logger.LogError(ctx, "Auth failed: token invalid", appErr)
				errors.AbortWith(c, appErr)
				return
			}

			token, _, err := authService.GetClient().DecodeAccessToken(
				c,
				accessToken,
				keycloakDataAccess.Realm,
			)

			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				logger.LogError(ctx, "Auth failed: decode token error", appError)
				errors.AbortWith(c, appError)
				return
			}

			claims := token.Claims.(jsonToken.MapClaims)

			jsonData, _ := json.Marshal(claims)

			var userClaim entities.JWTClaim

			err = json.Unmarshal(jsonData, &userClaim)
			if err != nil {
				appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
				logger.LogError(ctx, "Auth failed: unmarshal claims error", appError)
				errors.AbortWith(c, appError)
				return
			}

			// Extract roles from realm_access instead of resource_access
			// This works for both public and confidential clients
			if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
				if roles, ok := realmAccess["roles"].([]interface{}); ok {
					rolesBytes, _ := json.Marshal(roles)
					err = json.Unmarshal(rolesBytes, &userClaim.Roles)
					if err != nil {
						appError := errors.NewAppError(entities.ErrMiddleware, err.Error(), nil, err)
						logger.LogError(ctx, "Auth failed: unmarshal roles error", appError)
						errors.AbortWith(c, appError)
						return
					}
				}
			}

			// Check if user has at least one of the required roles. A valid token
			// that lacks any required role is an authorization failure (403), handled
			// by authorizeRole below; 401 is reserved for a missing/invalid/expired
			// token (handled above).
			matchedRole, authorized := authorizeRole(c, logger, userClaim.Roles, roles)
			if !authorized {
				return
			}

			logger.Info(ctx, "Auth success", map[string]interface{}{
				"request_id":     requestID,
				"ip":             c.ClientIP(),
				"matched_role":   matchedRole,
				"required_roles": roles,
				"user_roles":     userClaim.Roles,
				"user_id":        userClaim.ID,
				"email":          userClaim.Email,
			})

			// Set claims and individual user data for easy access
			c.Set("claims", userClaim)
			c.Set("user_id", userClaim.ID.String())
			c.Set("user_username", userClaim.Username)
			c.Set("user_email", userClaim.Email)
			c.Set("user_role", matchedRole) // The role that matched
			c.Set("user_roles", userClaim.Roles)
			if userClaim.OrganizationID != nil {
				c.Set("organization_id", *userClaim.OrganizationID)
			}

			// Check subscription status after authentication
			subscriptionMiddleware.CheckSubscription()(c)

			// If subscription check aborted, don't call handler
			if c.IsAborted() {
				return
			}

			handler(c)
		}
	}
}

// authorizeRole checks whether userRoles contains at least one of the required
// roles. On success it returns the matched role and true. On failure it writes the
// standard 403 envelope (code "insufficient_role") via errors.AbortWith and returns
// ("", false). It is a 403 (not 401) because the token is valid; the caller merely
// lacks permission. Extracted so the authorization decision is unit-testable without
// a live Keycloak token.
func authorizeRole(c *gin.Context, log logger.Logger, userRoles types.Array, required []string) (string, bool) {
	for _, requiredRole := range required {
		if userRoles.Contains(requiredRole) {
			return requiredRole, true
		}
	}

	ctx := c.Request.Context()
	log.Info(ctx, "Role check failed", map[string]interface{}{
		"required_roles": required,
		"user_roles":     userRoles,
	})
	forbidden := errors.Forbidden("insufficient_role", "Você não tem permissão para realizar esta ação")
	log.LogError(ctx, "Auth failed: missing required role", forbidden)
	errors.AbortWith(c, forbidden)
	return "", false
}
