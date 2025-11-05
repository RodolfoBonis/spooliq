package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/RodolfoBonis/spooliq/core/types"
	"github.com/gin-gonic/gin"
)

// GetOrganizationID extracts organization_id from Gin context
// Returns empty string if not found
func GetOrganizationID(c *gin.Context) string {
	if orgID, exists := c.Get("organization_id"); exists {
		if orgIDStr, ok := orgID.(string); ok {
			return orgIDStr
		}
	}
	return ""
}

// GetUserID extracts user_id from Gin context
// Returns empty string if not found
func GetUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	return ""
}

// GetUserEmail extracts user email from Gin context
// Returns empty string if not found
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get("user_email"); exists {
		if emailStr, ok := email.(string); ok {
			return emailStr
		}
	}
	return ""
}

// IsAdmin checks if the user has admin role
func IsAdmin(c *gin.Context) bool {
	if role, exists := c.Get("user_role"); exists {
		if roleStr, ok := role.(string); ok {
			return roleStr == "admin"
		}
	}
	return false
}

// IsPlatformAdmin checks if user has PlatformAdmin role
func IsPlatformAdmin(c *gin.Context) bool {
	roles, exists := c.Get("user_roles")
	if !exists {
		return false
	}

	// Try types.Array first (from JWT claims)
	if rolesArray, ok := roles.(types.Array); ok {
		return rolesArray.Contains("PlatformAdmin")
	}

	// Fallback to []string
	if rolesSlice, ok := roles.([]string); ok {
		for _, role := range rolesSlice {
			if role == "PlatformAdmin" {
				return true
			}
		}
	}

	return false
}

// GetOrganizationIDString returns organization_id as string (alias for GetOrganizationID)
// Provided for backward compatibility and explicit naming
func GetOrganizationIDString(c *gin.Context) string {
	return GetOrganizationID(c)
}

// GetUserRoles extracts user roles from Gin context
// Returns empty slice if not found
func GetUserRoles(c *gin.Context) []string {
	if roles, exists := c.Get("user_roles"); exists {
		// Try types.Array first (from JWT claims)
		if rolesArray, ok := roles.(types.Array); ok {
			result := make([]string, 0, len(rolesArray))
			for _, role := range rolesArray {
				if roleStr, ok := role.(string); ok {
					result = append(result, roleStr)
				}
			}
			return result
		}

		// Fallback to []string
		if rolesSlice, ok := roles.([]string); ok {
			return rolesSlice
		}
	}
	return []string{}
}

// OrganizationData represents organization information from context
type OrganizationData struct {
	ID         string
	Name       string
	Attributes map[string][]string
}

// GetOrganizationFromContext extracts organization data from Gin context
func GetOrganizationFromContext(c *gin.Context) (*OrganizationData, error) {
	orgID := GetOrganizationID(c)
	if orgID == "" {
		return nil, errors.New("organization ID not found in context")
	}

	// Try to get organization data from context
	if org, exists := c.Get("organization"); exists {
		if orgData, ok := org.(*OrganizationData); ok {
			return orgData, nil
		}
	}

	// If not in context, create minimal organization data
	return &OrganizationData{
		ID:         orgID,
		Attributes: make(map[string][]string),
	}, nil
}

// GetCurrentTimeString returns current time as ISO 8601 string
func GetCurrentTimeString() string {
	return time.Now().Format(time.RFC3339)
}

// IntToString converts int to string
func IntToString(n int) string {
	return fmt.Sprintf("%d", n)
}
