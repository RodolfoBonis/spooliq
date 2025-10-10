package helpers

import (
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

	rolesSlice, ok := roles.([]string)
	if !ok {
		return false
	}

	for _, role := range rolesSlice {
		if role == "PlatformAdmin" {
			return true
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
		if rolesSlice, ok := roles.([]string); ok {
			return rolesSlice
		}
	}
	return []string{}
}
