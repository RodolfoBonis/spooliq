package services

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	companyEntities "github.com/RodolfoBonis/spooliq/features/company/domain/entities"
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

	// Fetch all active companies (not suspended or cancelled)
	companies, err := s.companyRepository.FindAllActive(ctx)
	if err != nil {
		s.logger.Error(ctx, "Failed to fetch active companies", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	s.logger.Info(ctx, "Checking subscriptions", map[string]interface{}{
		"company_count": len(companies),
	})

	// Check each company's subscription status
	checkedCount := 0
	errorCount := 0

	for _, company := range companies {
		// Skip platform companies
		if company.IsPlatformCompany {
			continue
		}

		if err := s.checkCompanySubscription(ctx, company); err != nil {
			s.logger.Error(ctx, "Failed to check company subscription", map[string]interface{}{
				"organization_id": company.OrganizationID,
				"company_name":    company.Name,
				"error":           err.Error(),
			})
			errorCount++
			continue
		}

		checkedCount++
	}

	s.logger.Info(ctx, "Daily subscription check completed", map[string]interface{}{
		"checked": checkedCount,
		"errors":  errorCount,
		"total":   len(companies),
	})

	return nil
}

// checkCompanySubscription checks a single company's subscription status
func (s *SubscriptionCheckerService) checkCompanySubscription(ctx context.Context, company *companyEntities.CompanyEntity) error {
	// TODO: Refactor to use PaymentGatewayLink table for Asaas integration
	// For now, this service is disabled pending refactoring to the new FK structure
	// where Asaas info is in payment_gateway_links, not directly in companies table

	// For trial companies
	if company.SubscriptionStatus == "trial" {
		if company.TrialEndsAt != nil && time.Now().After(*company.TrialEndsAt) {
			// Trial has ended - need to check payment gateway link
			// TODO: Query PaymentGatewayLinkRepository to get gateway info
			s.logger.Info(ctx, "Trial ended but payment gateway check not yet implemented", map[string]interface{}{
				"organization_id": company.OrganizationID,
			})
			// For now, just suspend the company
			if false { // Disabled - needs refactoring
				// No subscription created - suspend the account
				company.SubscriptionStatus = "suspended"
				if err := s.companyRepository.Update(ctx, company); err != nil {
					return err
				}
				s.logger.Info(ctx, "Company suspended - trial ended without subscription", map[string]interface{}{
					"organization_id": company.OrganizationID,
				})
			}
		}
	}

	// For active companies
	if company.SubscriptionStatus == "active" {
		// TODO: Query PaymentGatewayLinkRepository to get Asaas subscription ID
		// Then check with Asaas API
		s.logger.Info(ctx, "Active subscription check not yet implemented for new FK structure", map[string]interface{}{
			"organization_id": company.OrganizationID,
		})
	}

	// Note: LastPaymentCheck field removed - status changes now tracked via StatusUpdatedAt
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
