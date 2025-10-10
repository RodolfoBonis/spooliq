package services

import (
	"context"
	
	"github.com/RodolfoBonis/spooliq/core/logger"
)

// IEmailService defines the interface for email notifications
type IEmailService interface {
	SendTrialEndingEmail(ctx context.Context, email, companyName string, daysRemaining int) error
	SendSubscriptionSuspendedEmail(ctx context.Context, email, companyName string) error
	SendPaymentConfirmedEmail(ctx context.Context, email, companyName string, amount float64) error
	SendSubscriptionCancelledEmail(ctx context.Context, email, companyName string) error
}

// EmailService handles email notifications
type EmailService struct {
	logger logger.Logger
}

// NewEmailService creates a new email service
func NewEmailService(logger logger.Logger) IEmailService {
	return &EmailService{
		logger: logger,
	}
}

// SendTrialEndingEmail sends notification when trial is ending
func (s *EmailService) SendTrialEndingEmail(ctx context.Context, email, companyName string, daysRemaining int) error {
	s.logger.Info(ctx, "Sending trial ending email (placeholder)", map[string]interface{}{
		"email":          email,
		"company":        companyName,
		"days_remaining": daysRemaining,
	})
	// TODO: Implement actual email sending (SMTP, SendGrid, AWS SES, etc.)
	return nil
}

// SendSubscriptionSuspendedEmail sends notification when subscription is suspended
func (s *EmailService) SendSubscriptionSuspendedEmail(ctx context.Context, email, companyName string) error {
	s.logger.Info(ctx, "Sending subscription suspended email (placeholder)", map[string]interface{}{
		"email":   email,
		"company": companyName,
	})
	// TODO: Implement actual email sending
	return nil
}

// SendPaymentConfirmedEmail sends notification when payment is confirmed
func (s *EmailService) SendPaymentConfirmedEmail(ctx context.Context, email, companyName string, amount float64) error {
	s.logger.Info(ctx, "Sending payment confirmed email (placeholder)", map[string]interface{}{
		"email":   email,
		"company": companyName,
		"amount":  amount,
	})
	// TODO: Implement actual email sending
	return nil
}

// SendSubscriptionCancelledEmail sends notification when subscription is cancelled
func (s *EmailService) SendSubscriptionCancelledEmail(ctx context.Context, email, companyName string) error {
	s.logger.Info(ctx, "Sending subscription cancelled email (placeholder)", map[string]interface{}{
		"email":   email,
		"company": companyName,
	})
	// TODO: Implement actual email sending
	return nil
}

