package entities

import "time"

// CompanyListItem represents a company in the list response
type CompanyListItem struct {
	ID                  string     `json:"id"`
	OrganizationID      string     `json:"organization_id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	SubscriptionStatus  string     `json:"subscription_status"`
	SubscriptionPlan    string     `json:"subscription_plan"`
	IsPlatformCompany   bool       `json:"is_platform_company"`
	TrialEndsAt         *time.Time `json:"trial_ends_at,omitempty"`
	NextPaymentDue      *time.Time `json:"next_payment_due,omitempty"`
	AsaasCustomerID     string     `json:"asaas_customer_id"`
	AsaasSubscriptionID string     `json:"asaas_subscription_id"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ListCompaniesResponse represents the paginated list response
type ListCompaniesResponse struct {
	Companies  []CompanyListItem `json:"companies"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalCount int64             `json:"total_count"`
	TotalPages int               `json:"total_pages"`
}

// CompanyDetailsResponse represents detailed company information
type CompanyDetailsResponse struct {
	ID                    string     `json:"id"`
	OrganizationID        string     `json:"organization_id"`
	Name                  string     `json:"name"`
	Email                 string     `json:"email"`
	Phone                 string     `json:"phone"`
	WhatsApp              string     `json:"whatsapp"`
	Instagram             string     `json:"instagram"`
	Website               string     `json:"website"`
	LogoURL               string     `json:"logo_url"`
	SubscriptionStatus    string     `json:"subscription_status"`
	SubscriptionPlan      string     `json:"subscription_plan"`
	IsPlatformCompany     bool       `json:"is_platform_company"`
	TrialEndsAt           *time.Time `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at,omitempty"`
	AsaasCustomerID       string     `json:"asaas_customer_id"`
	AsaasSubscriptionID   string     `json:"asaas_subscription_id"`
	LastPaymentCheck      *time.Time `json:"last_payment_check,omitempty"`
	NextPaymentDue        *time.Time `json:"next_payment_due,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// UpdateStatusRequest represents request to update subscription status
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=trial active suspended cancelled permanent"`
	Reason string `json:"reason" validate:"required"`
}

// UpdatePlanRequest represents request to update subscription plan
type UpdatePlanRequest struct {
	Plan   string `json:"plan" validate:"required,oneof=basic pro enterprise"`
	Reason string `json:"reason" validate:"required"`
}

// SubscriptionListItem represents a subscription in the list
type SubscriptionListItem struct {
	OrganizationID      string     `json:"organization_id"`
	CompanyName         string     `json:"company_name"`
	SubscriptionStatus  string     `json:"subscription_status"`
	SubscriptionPlan    string     `json:"subscription_plan"`
	AsaasSubscriptionID string     `json:"asaas_subscription_id"`
	NextPaymentDue      *time.Time `json:"next_payment_due,omitempty"`
	TrialEndsAt         *time.Time `json:"trial_ends_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// ListSubscriptionsResponse represents paginated subscriptions list
type ListSubscriptionsResponse struct {
	Subscriptions []SubscriptionListItem `json:"subscriptions"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
	TotalCount    int64                  `json:"total_count"`
	TotalPages    int                    `json:"total_pages"`
}

// PaymentHistoryItem represents a single payment record
type PaymentHistoryItem struct {
	ID             string     `json:"id"`
	AsaasPaymentID string     `json:"asaas_payment_id"`
	Amount         float64    `json:"amount"`
	Status         string     `json:"status"`
	DueDate        time.Time  `json:"due_date"`
	PaymentDate    *time.Time `json:"payment_date,omitempty"`
	InvoiceURL     string     `json:"invoice_url"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PaymentHistoryResponse represents payment history
type PaymentHistoryResponse struct {
	Payments   []PaymentHistoryItem `json:"payments"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalCount int64                `json:"total_count"`
	TotalPages int                  `json:"total_pages"`
}

// SubscriptionDetailsResponse represents detailed subscription info
type SubscriptionDetailsResponse struct {
	OrganizationID        string               `json:"organization_id"`
	CompanyName           string               `json:"company_name"`
	SubscriptionStatus    string               `json:"subscription_status"`
	SubscriptionPlan      string               `json:"subscription_plan"`
	AsaasCustomerID       string               `json:"asaas_customer_id"`
	AsaasSubscriptionID   string               `json:"asaas_subscription_id"`
	TrialEndsAt           *time.Time           `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time           `json:"subscription_started_at,omitempty"`
	NextPaymentDue        *time.Time           `json:"next_payment_due,omitempty"`
	LastPaymentCheck      *time.Time           `json:"last_payment_check,omitempty"`
	RecentPayments        []PaymentHistoryItem `json:"recent_payments"`
}

// CancelSubscriptionRequest represents request to cancel subscription
type CancelSubscriptionRequest struct {
	Reason string `json:"reason" validate:"required"`
}
