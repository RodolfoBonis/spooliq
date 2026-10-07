package entities

import (
	"time"

	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// CompanyEntity represents a company in the domain layer
type CompanyEntity struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	TradeName      *string   `json:"trade_name,omitempty"`
	Document       *string   `json:"document,omitempty"` // CNPJ
	Email          *string   `json:"email,omitempty"`
	Phone          *string   `json:"phone,omitempty"`
	WhatsApp       *string   `json:"whatsapp,omitempty"`
	Instagram      *string   `json:"instagram,omitempty"`
	Website        *string   `json:"website,omitempty"`
	LogoURL        *string   `json:"logo_url,omitempty"`
	Address        *string   `json:"address,omitempty"`
	City           *string   `json:"city,omitempty"`
	State          *string   `json:"state,omitempty"`
	ZipCode        *string   `json:"zip_code,omitempty"`

	// DefaultTaxRate is the organization's default "por dentro" tax rate (percent).
	DefaultTaxRate float64 `json:"default_tax_rate"`

	// DefaultQuoteValidityDays is the default number of days a sent quote stays valid.
	DefaultQuoteValidityDays int `json:"default_quote_validity_days"`
	// DefaultPaymentTerms is the organization's default payment terms.
	DefaultPaymentTerms *string `json:"default_payment_terms,omitempty"`

	// Subscription fields
	SubscriptionStatus    string                                       `json:"subscription_status"`
	SubscriptionPlanID    *uuid.UUID                                   `json:"subscription_plan_id,omitempty"` // FK to subscription_plans
	CurrentPlan           *subscriptionEntities.SubscriptionPlanEntity `json:"current_plan,omitempty"`
	StatusUpdatedAt       time.Time                                    `json:"status_updated_at"`
	IsPlatformCompany     bool                                         `json:"is_platform_company"`
	TrialEndsAt           *time.Time                                   `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time                                   `json:"subscription_started_at,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
