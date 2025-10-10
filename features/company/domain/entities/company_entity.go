package entities

import (
	"time"

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

	// Subscription fields
	SubscriptionStatus    string     `json:"subscription_status"`
	IsPlatformCompany     bool       `json:"is_platform_company"`
	TrialEndsAt           *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at,omitempty"`
	SubscriptionPlan      string     `json:"subscription_plan"`
	AsaasCustomerID       string     `json:"asaas_customer_id,omitempty"`
	AsaasSubscriptionID   string     `json:"asaas_subscription_id,omitempty"`
	LastPaymentCheck      *time.Time `json:"last_payment_check,omitempty"`
	NextPaymentDue        *time.Time `json:"next_payment_due,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
