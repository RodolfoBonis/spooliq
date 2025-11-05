package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionMiddleware handles subscription status checking
type SubscriptionMiddleware struct {
	companyRepository      companyRepositories.CompanyRepository
	subscriptionRepository subscriptionRepositories.SubscriptionRepository
	logger                 logger.Logger
}

// NewSubscriptionMiddleware creates a new subscription middleware
func NewSubscriptionMiddleware(
	companyRepository companyRepositories.CompanyRepository,
	subscriptionRepository subscriptionRepositories.SubscriptionRepository,
	logger logger.Logger,
) *SubscriptionMiddleware {
	return &SubscriptionMiddleware{
		companyRepository:      companyRepository,
		subscriptionRepository: subscriptionRepository,
		logger:                 logger,
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
					"error":               "Trial period has expired",
					"subscription_status": "trial_expired",
					"trial_ended_at":      company.TrialEndsAt.Format("2006-01-02T15:04:05Z07:00"),
					"message":             "Please subscribe to continue using the service",
				})
				c.Abort()
				return
			}
			// Trial is still active, allow access
			c.Next()
			return

		case "active", "permanent", "ACTIVE", "PERMANENT":
			// Subscription is active, but verify payment status
			hasValidPayment, err := m.hasValidPayment(ctx, organizationID)
			if err != nil {
				m.logger.Error(ctx, "Failed to verify payment status", map[string]interface{}{
					"error":           err.Error(),
					"organization_id": organizationID,
				})
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Unable to verify subscription status",
					"message": "Please contact support",
				})
				c.Abort()
				return
			}

			if !hasValidPayment {
				// Check if this is a payment recovery endpoint
				if isPaymentRecoveryEndpoint(c.Request.URL.Path) {
					m.logger.Info(ctx, "Payment recovery access granted", map[string]interface{}{
						"organization_id": organizationID,
						"path":            c.Request.URL.Path,
						"status":          company.SubscriptionStatus,
						"reason":          "payment_pending_recovery_access",
					})
					c.Next()
					return
				}

				// Block access to non-payment-recovery endpoints with helpful guidance
				m.logger.Error(ctx, "Access denied: active subscription but no valid payment", map[string]interface{}{
					"organization_id": organizationID,
					"status":          company.SubscriptionStatus,
					"path":            c.Request.URL.Path,
				})
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error":               "Payment required to access this feature",
					"subscription_status": "payment_pending",
					"message":             "Your subscription is active but payment is still being processed. You can still access payment-related endpoints to resolve this issue.",
					"payment_recovery_endpoints": []string{
						"/v1/payment-methods",
						"/v1/subscriptions/subscribe",
						"/v1/subscriptions/status",
					},
					"help": "You can add payment methods, retry subscription, or check status while payment is being processed.",
				})
				c.Abort()
				return
			}

			// Valid payment found, allow access
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
				// Note: next_payment_due removed - calculate dynamically from SubscriptionPayments if needed
				"message": "Please update your payment information to reactivate your subscription",
			})
			c.Abort()
			return

		case "cancelled", "CANCELLED":
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
		"/v1/subscriptions/plans", // Public endpoint - anyone can view available plans
	}

	for _, endpoint := range publicEndpoints {
		if len(path) >= len(endpoint) && path[:len(endpoint)] == endpoint {
			return true
		}
	}

	return false
}

// isPaymentRecoveryEndpoint checks if the endpoint should remain accessible during payment recovery
func isPaymentRecoveryEndpoint(path string) bool {
	paymentRecoveryEndpoints := []string{
		"/v1/payment-methods",         // All payment method operations
		"/v1/subscriptions/subscribe", // Retry subscription
		"/v1/subscriptions/status",    // Check subscription status
		"/v1/subscriptions/plans",     // View available plans (already public but for consistency)
	}

	for _, endpoint := range paymentRecoveryEndpoints {
		if len(path) >= len(endpoint) && path[:len(endpoint)] == endpoint {
			return true
		}
	}

	return false
}

// hasValidPayment checks if the organization has at least one confirmed/received payment
func (m *SubscriptionMiddleware) hasValidPayment(ctx context.Context, organizationID string) (bool, error) {
	// Convert string organizationID to UUID
	orgUUID, err := uuid.Parse(organizationID)
	if err != nil {
		return false, err
	}

	// Get latest subscription payments for the organization (limit 10 to check recent payments)
	payments, err := m.subscriptionRepository.FindAll(ctx, orgUUID, 10, 0)
	if err != nil {
		return false, err
	}

	// Check if there's at least one confirmed or received payment
	for _, payment := range payments {
		if payment.Status == "confirmed" || payment.Status == "received" || payment.Status == "anticipated" {
			return true, nil
		}
	}

	return false, nil
}
