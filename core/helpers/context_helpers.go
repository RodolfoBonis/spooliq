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

