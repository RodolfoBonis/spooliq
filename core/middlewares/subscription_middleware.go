package middlewares

import (
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/gin-gonic/gin"
)

// SubscriptionMiddleware handles subscription status checking
type SubscriptionMiddleware struct {
	companyRepository companyRepositories.CompanyRepository
	logger            logger.Logger
}

// NewSubscriptionMiddleware creates a new subscription middleware
func NewSubscriptionMiddleware(
	companyRepository companyRepositories.CompanyRepository,
	logger logger.Logger,
) *SubscriptionMiddleware {
	return &SubscriptionMiddleware{
		companyRepository: companyRepository,
		logger:            logger,
	}
}

// CheckSubscription verifies if the company's subscription is active
func (m *SubscriptionMiddleware) CheckSubscription() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Skip check for public endpoints
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Skip check for PlatformAdmin users
		if helpers.IsPlatformAdmin(c) {
			m.logger.Info(ctx, "Subscription check skipped for PlatformAdmin", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			c.Next()
			return
		}

		// Get organization_id from context (set by auth middleware)
		organizationID := helpers.GetOrganizationIDString(c)
		if organizationID == "" {
			m.logger.Error(ctx, "Organization ID not found in subscription middleware", map[string]interface{}{
				"path": c.Request.URL.Path,
			})
			appError := errors.NewAppError(entities.ErrUnauthorized, "Organization ID required", nil, nil)
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			c.Abort()
			return
		}

		// Fetch company from database
		company, err := m.companyRepository.FindByOrganizationID(ctx, organizationID)
		if err != nil {
			m.logger.Error(ctx, "Failed to fetch company for subscription check", map[string]interface{}{
				"error":           err.Error(),
				"organization_id": organizationID,
			})
			appError := errors.NewAppError(entities.ErrDatabase, "Failed to verify subscription status", nil, err)
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			c.Abort()
			return
		}

		// Skip check if this is a platform company (never expires)
		if company.IsPlatformCompany {
			m.logger.Info(ctx, "Subscription check skipped for platform company", map[string]interface{}{
				"organization_id": organizationID,
				"company_name":    company.Name,
			})
			c.Next()
			return
		}

		// Check subscription status
		switch company.SubscriptionStatus {
		case "trial":
			// Check if trial has expired
			if company.TrialEndsAt != nil && time.Now().After(*company.TrialEndsAt) {
				m.logger.Error(ctx, "Trial expired", map[string]interface{}{
					"organization_id": organizationID,
					"trial_ends_at":   company.TrialEndsAt,
				})
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error":            "Trial period has expired",
					"subscription_status": "trial_expired",
					"trial_ended_at":   company.TrialEndsAt.Format("2006-01-02T15:04:05Z07:00"),
					"message":          "Please subscribe to continue using the service",
				})
				c.Abort()
				return
			}
			// Trial is still active, allow access
			c.Next()
			return

		case "active", "permanent":
			// Subscription is active, allow access
			c.Next()
			return

		case "suspended":
			m.logger.Error(ctx, "Access denied: subscription suspended", map[string]interface{}{
				"organization_id": organizationID,
				"status":          company.SubscriptionStatus,
			})
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":               "Subscription suspended due to payment issues",
				"subscription_status": "suspended",
				"next_payment_due":    formatTimePtr(company.NextPaymentDue),
				"message":             "Please update your payment information to reactivate your subscription",
			})
			c.Abort()
			return

		case "cancelled":
			m.logger.Error(ctx, "Access denied: subscription cancelled", map[string]interface{}{
				"organization_id": organizationID,
				"status":          company.SubscriptionStatus,
			})
			c.JSON(http.StatusForbidden, gin.H{
				"error":               "Subscription has been cancelled",
				"subscription_status": "cancelled",
				"message":             "Please contact support to reactivate your account",
			})
			c.Abort()
			return

		default:
			m.logger.Error(ctx, "Unknown subscription status", map[string]interface{}{
				"organization_id": organizationID,
				"status":          company.SubscriptionStatus,
			})
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Unable to verify subscription status",
				"message": "Please contact support",
			})
			c.Abort()
			return
		}
	}
}

// isPublicEndpoint checks if the endpoint is public (no subscription check needed)
func isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/v1/register",
		"/v1/login",
		"/v1/logout",
		"/v1/refresh",
		"/health",
		"/metrics",
		"/docs",
		"/swagger",
		"/v1/webhooks",
	}

	for _, endpoint := range publicEndpoints {
		if len(path) >= len(endpoint) && path[:len(endpoint)] == endpoint {
			return true
		}
	}

	return false
}

// formatTimePtr safely formats a time pointer, returns empty string if nil
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

