package services

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
)

// SubscriptionCheckerService performs daily subscription status checks
type SubscriptionCheckerService struct {
	companyRepository companyRepositories.CompanyRepository
	asaasService      IAsaasService
	logger            logger.Logger
}

// NewSubscriptionCheckerService creates a new subscription checker service
func NewSubscriptionCheckerService(
	companyRepository companyRepositories.CompanyRepository,
	asaasService IAsaasService,
	logger logger.Logger,
) *SubscriptionCheckerService {
	return &SubscriptionCheckerService{
		companyRepository: companyRepository,
		asaasService:      asaasService,
		logger:            logger,
	}
}

// CheckAllSubscriptions runs the daily subscription check for all companies
func (s *SubscriptionCheckerService) CheckAllSubscriptions(ctx context.Context) error {
	s.logger.Info(ctx, "Starting daily subscription check", nil)
	
	// This is a placeholder implementation that logs actions
	// In production, you would:
	// 1. Call s.companyRepository to get all companies with status "trial" or "active"
	// 2. For each company:
	//    - Skip if IsPlatformCompany = true
	//    - For trial: check if TrialEndsAt < now
	//      * If expired, call s.asaasService.GetSubscription(company.AsaasSubscriptionID)
	//      * Check if first payment was made
	//      * Update status to "active" if payment confirmed, "suspended" if not
	//    - For active: call s.asaasService.GetSubscription()
	//      * Check for overdue or cancelled status
	//      * Update company status accordingly
	// 3. Update company in database
	// 4. Send email notifications via EmailService
	
	s.logger.Info(ctx, "Daily subscription check completed", map[string]interface{}{
		"note": "This is a placeholder. Implement full logic before production.",
	})
	
	return nil
}

// checkCompanySubscription checks a single company's subscription status
func (s *SubscriptionCheckerService) checkCompanySubscription(ctx context.Context, company interface{}) error {
	// Type assertion to get company entity
	// This should be implemented based on your company entity structure

	// For trial companies
	// if company.SubscriptionStatus == "trial" {
	//     if company.TrialEndsAt != nil && time.Now().After(*company.TrialEndsAt) {
	//         // Check Asaas subscription status
	//         // If subscription active and first payment confirmed, update to 'active'
	//         // If no payment, update to 'suspended', send notification email
	//     }
	// }

	// For active companies
	// if company.SubscriptionStatus == "active" {
	//     // Check Asaas subscription status
	//     // If overdue, update to 'suspended'
	//     // If cancelled in Asaas, update to 'cancelled'
	// }

	// TODO: Implement full subscription checking logic
	// This is a placeholder implementation
	s.logger.Info(ctx, "Checking company subscription", map[string]interface{}{
		"company": "placeholder",
	})

	return nil
}

// StartDailyChecker starts the daily subscription checker (runs at 3 AM)
func (s *SubscriptionCheckerService) StartDailyChecker(ctx context.Context) {
	// Calculate time until next 3 AM
	now := time.Now()
	next3AM := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if now.After(next3AM) {
		next3AM = next3AM.Add(24 * time.Hour)
	}

	duration := time.Until(next3AM)

	s.logger.Info(ctx, "Scheduling daily subscription checker", map[string]interface{}{
		"next_run": next3AM.Format("2006-01-02 15:04:05"),
		"duration": duration.String(),
	})

	// Initial timer until 3 AM
	timer := time.NewTimer(duration)

	go func() {
		for {
			select {
			case <-timer.C:
				// Run the check
				if err := s.CheckAllSubscriptions(ctx); err != nil {
					s.logger.Error(ctx, "Daily subscription check failed", map[string]interface{}{
						"error": err.Error(),
					})
				}

				// Reset timer for next day (24 hours)
				timer.Reset(24 * time.Hour)

			case <-ctx.Done():
				s.logger.Info(ctx, "Stopping daily subscription checker", nil)
				timer.Stop()
				return
			}
		}
	}()
}
