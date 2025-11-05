package entities

import (
	"time"

	subscriptionEntities "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
)

// CompanyListItem represents a company in the list response
type CompanyListItem struct {
	ID                 string                                         `json:"id"`
	OrganizationID     string                                         `json:"organization_id"`
	Name               string                                         `json:"name"`
	Email              string                                         `json:"email"`
	SubscriptionStatus string                                         `json:"subscription_status"`
	SubscriptionPlanID *string                                        `json:"subscription_plan_id,omitempty"` // UUID as string
	CurrentPlan        *subscriptionEntities.SubscriptionPlanResponse `json:"current_plan,omitempty"`
	IsPlatformCompany  bool                                           `json:"is_platform_company"`
	TrialEndsAt        *time.Time                                     `json:"trial_ends_at,omitempty"`
	CreatedAt          time.Time                                      `json:"created_at"`
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
	ID                    string                                         `json:"id"`
	OrganizationID        string                                         `json:"organization_id"`
	Name                  string                                         `json:"name"`
	Email                 string                                         `json:"email"`
	Phone                 string                                         `json:"phone"`
	WhatsApp              string                                         `json:"whatsapp"`
	Instagram             string                                         `json:"instagram"`
	Website               string                                         `json:"website"`
	LogoURL               string                                         `json:"logo_url"`
	SubscriptionStatus    string                                         `json:"subscription_status"`
	SubscriptionPlanID    *string                                        `json:"subscription_plan_id,omitempty"` // UUID as string, FK to subscription_plans
	CurrentPlan           *subscriptionEntities.SubscriptionPlanResponse `json:"current_plan,omitempty"`
	StatusUpdatedAt       *time.Time                                     `json:"status_updated_at,omitempty"`
	IsPlatformCompany     bool                                           `json:"is_platform_company"`
	TrialEndsAt           *time.Time                                     `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time                                     `json:"subscription_started_at,omitempty"`
	// Note: Asaas info moved to PaymentGatewayLink table
	// AsaasCustomerID and AsaasSubscriptionID can be queried from payment_gateway_links if needed
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	OrganizationID     string                                         `json:"organization_id"`
	CompanyName        string                                         `json:"company_name"`
	SubscriptionStatus string                                         `json:"subscription_status"`
	SubscriptionPlanID *string                                        `json:"subscription_plan_id,omitempty"` // UUID as string
	CurrentPlan        *subscriptionEntities.SubscriptionPlanResponse `json:"current_plan,omitempty"`
	TrialEndsAt        *time.Time                                     `json:"trial_ends_at,omitempty"`
	CreatedAt          time.Time                                      `json:"created_at"`
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
	OrganizationID        string                                         `json:"organization_id"`
	CompanyName           string                                         `json:"company_name"`
	SubscriptionStatus    string                                         `json:"subscription_status"`
	SubscriptionPlanID    *string                                        `json:"subscription_plan_id,omitempty"` // UUID as string
	CurrentPlan           *subscriptionEntities.SubscriptionPlanResponse `json:"current_plan,omitempty"`
	StatusUpdatedAt       *time.Time                                     `json:"status_updated_at,omitempty"`
	TrialEndsAt           *time.Time                                     `json:"trial_ends_at,omitempty"`
	SubscriptionStartedAt *time.Time                                     `json:"subscription_started_at,omitempty"`
	// Note: Asaas info and payment timing moved to separate tables
	// Use PaymentGatewayLink and SubscriptionPayments for detailed info
	RecentPayments []PaymentHistoryItem `json:"recent_payments"`
}

// CancelSubscriptionRequest represents request to cancel subscription
type CancelSubscriptionRequest struct {
	Reason string `json:"reason" validate:"required"`
}

// AdminStats represents platform analytics and metrics
type AdminStats struct {
	TotalCompanies       int     `json:"total_companies"`
	ActiveSubscriptions  int     `json:"active_subscriptions"`
	TrialSubscriptions   int     `json:"trial_subscriptions"`
	OverdueSubscriptions int     `json:"overdue_subscriptions"`
	TotalMRR             float64 `json:"total_mrr"`  // Monthly Recurring Revenue in cents
	ChurnRate            float64 `json:"churn_rate"` // percentage
}

// PlanStats represents statistics for a specific subscription plan
type PlanStats struct {
	PlanID           string  `json:"plan_id"`
	PlanName         string  `json:"plan_name"`
	TotalCompanies   int     `json:"total_companies"`
	ActiveCompanies  int     `json:"active_companies"`
	TrialCompanies   int     `json:"trial_companies"`
	TotalActiveUsers int     `json:"total_active_users"`
	MonthlyRevenue   float64 `json:"monthly_revenue"`
	AnnualRevenue    float64 `json:"annual_revenue"`
	ChurnRate        float64 `json:"churn_rate"`
	ConversionRate   float64 `json:"conversion_rate"` // trial to paid conversion
}

// PlanCompanyListItem represents a company using a specific plan
type PlanCompanyListItem struct {
	ID                    string     `json:"id"`
	OrganizationID        string     `json:"organization_id"`
	Name                  string     `json:"name"`
	Email                 string     `json:"email"`
	SubscriptionStatus    string     `json:"subscription_status"`
	SubscriptionStartedAt *time.Time `json:"subscription_started_at,omitempty"`
	TrialEndsAt           *time.Time `json:"trial_ends_at,omitempty"`
	LastPaymentDate       *time.Time `json:"last_payment_date,omitempty"`
	NextPaymentDate       *time.Time `json:"next_payment_date,omitempty"`
	TotalUsers            int        `json:"total_users"`
	CreatedAt             time.Time  `json:"created_at"`
}

// ListPlanCompaniesResponse represents companies using a specific plan
type ListPlanCompaniesResponse struct {
	Companies  []PlanCompanyListItem `json:"companies"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalCount int64                 `json:"total_count"`
	TotalPages int                   `json:"total_pages"`
}

// PlanFinancialReport represents financial metrics for a specific plan
type PlanFinancialReport struct {
	PlanID        string                  `json:"plan_id"`
	PlanName      string                  `json:"plan_name"`
	ReportPeriod  string                  `json:"report_period"` // "monthly", "quarterly", "yearly"
	Revenue       PlanRevenueMetrics      `json:"revenue"`
	Subscriptions PlanSubscriptionMetrics `json:"subscriptions"`
	Projections   PlanRevenueProjections  `json:"projections"`
	Trends        []PlanRevenueTrendPoint `json:"trends"`
}

// PlanRevenueMetrics represents revenue data for a plan
type PlanRevenueMetrics struct {
	CurrentPeriod    float64 `json:"current_period"`
	PreviousPeriod   float64 `json:"previous_period"`
	GrowthPercentage float64 `json:"growth_percentage"`
	AveragePerUser   float64 `json:"average_per_user"`
	TotalLifetime    float64 `json:"total_lifetime"`
}

// PlanSubscriptionMetrics represents subscription data for a plan
type PlanSubscriptionMetrics struct {
	NewSubscriptions       int     `json:"new_subscriptions"`
	CancelledSubscriptions int     `json:"cancelled_subscriptions"`
	ChurnRate              float64 `json:"churn_rate"`
	RetentionRate          float64 `json:"retention_rate"`
	ConversionRate         float64 `json:"conversion_rate"`
}

// PlanRevenueProjections represents revenue projections for a plan
type PlanRevenueProjections struct {
	NextMonth   float64 `json:"next_month"`
	NextQuarter float64 `json:"next_quarter"`
	NextYear    float64 `json:"next_year"`
	Methodology string  `json:"methodology"` // How the projection was calculated
}

// PlanRevenueTrendPoint represents a point in revenue trend
type PlanRevenueTrendPoint struct {
	Period        string  `json:"period"` // "2024-01", "2024-Q1", etc
	Revenue       float64 `json:"revenue"`
	Subscriptions int     `json:"subscriptions"`
}

// PlanDeletionCheck represents validation for plan deletion
type PlanDeletionCheck struct {
	CanDelete       bool     `json:"can_delete"`
	Reason          string   `json:"reason"`
	ActiveCompanies int      `json:"active_companies"`
	TrialCompanies  int      `json:"trial_companies"`
	BlockingIssues  []string `json:"blocking_issues,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// BulkOperationRequest represents a bulk operation request
type BulkOperationRequest struct {
	PlanIDs []string `json:"plan_ids" validate:"required,min=1"`
	Reason  string   `json:"reason" validate:"required"`
}

// BulkUpdateRequest represents a bulk update request
type BulkUpdateRequest struct {
	PlanIDs []string               `json:"plan_ids" validate:"required,min=1"`
	Updates map[string]interface{} `json:"updates" validate:"required"`
	Reason  string                 `json:"reason" validate:"required"`
}

// BulkOperationResult represents the result of a bulk operation
type BulkOperationResult struct {
	TotalRequested int                 `json:"total_requested"`
	Successful     int                 `json:"successful"`
	Failed         int                 `json:"failed"`
	Results        []BulkOperationItem `json:"results"`
	Summary        string              `json:"summary"`
}

// BulkOperationItem represents the result of an operation on a single plan
type BulkOperationItem struct {
	PlanID   string `json:"plan_id"`
	PlanName string `json:"plan_name"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// PlanAuditEntry represents an audit entry for plan changes
type PlanAuditEntry struct {
	ID        string                 `json:"id"`
	PlanID    string                 `json:"plan_id"`
	PlanName  string                 `json:"plan_name"`
	Action    string                 `json:"action"` // created, updated, deleted, activated, deactivated
	UserID    string                 `json:"user_id"`
	UserEmail string                 `json:"user_email"`
	Changes   map[string]interface{} `json:"changes"`
	Reason    string                 `json:"reason"`
	CreatedAt time.Time              `json:"created_at"`
}

// PlanAuditResponse represents paginated audit history
type PlanAuditResponse struct {
	Entries    []PlanAuditEntry `json:"entries"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalCount int64            `json:"total_count"`
	TotalPages int              `json:"total_pages"`
}

// PlanTemplate represents a plan template
type PlanTemplate struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	PlanData    PlanTemplateData `json:"plan_data"`
	IsActive    bool             `json:"is_active"`
	UsageCount  int              `json:"usage_count"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// PlanTemplateData represents the data structure for a plan template
type PlanTemplateData struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Price       float64               `json:"price"`
	Cycle       string                `json:"cycle"`
	Features    []PlanTemplateFeature `json:"features"`
}

// PlanTemplateFeature represents a feature in a plan template
type PlanTemplateFeature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
}

// CreateFromTemplateRequest represents request to create plan from template
type CreateFromTemplateRequest struct {
	TemplateID     string                 `json:"template_id" validate:"required"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
	Reason         string                 `json:"reason" validate:"required"`
}

// AvailableFeature represents a feature available for plans
type AvailableFeature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsActive    bool   `json:"is_active"`
}

// FeatureValidationRequest represents request to validate features
type FeatureValidationRequest struct {
	Features []PlanTemplateFeature `json:"features" validate:"required,min=1"`
}

// FeatureValidationResult represents result of feature validation
type FeatureValidationResult struct {
	IsValid         bool                  `json:"is_valid"`
	ValidFeatures   []PlanTemplateFeature `json:"valid_features"`
	InvalidFeatures []InvalidFeatureError `json:"invalid_features"`
	Suggestions     []string              `json:"suggestions,omitempty"`
}

// InvalidFeatureError represents an invalid feature error
type InvalidFeatureError struct {
	Feature PlanTemplateFeature `json:"feature"`
	Error   string              `json:"error"`
}

// PlanMigrationRequest represents request to migrate companies from one plan to another
type PlanMigrationRequest struct {
	FromPlanID   string                 `json:"from_plan_id" validate:"required"`
	ToPlanID     string                 `json:"to_plan_id" validate:"required"`
	CompanyIDs   []string               `json:"company_ids,omitempty"`   // If empty, migrates all companies
	ScheduledFor *time.Time             `json:"scheduled_for,omitempty"` // If empty, immediate
	NotifyUsers  bool                   `json:"notify_users"`
	Reason       string                 `json:"reason" validate:"required"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// PlanMigrationResult represents result of plan migration
type PlanMigrationResult struct {
	MigrationID    string                 `json:"migration_id"`
	FromPlanID     string                 `json:"from_plan_id"`
	FromPlanName   string                 `json:"from_plan_name"`
	ToPlanID       string                 `json:"to_plan_id"`
	ToPlanName     string                 `json:"to_plan_name"`
	TotalCompanies int                    `json:"total_companies"`
	Successful     int                    `json:"successful"`
	Failed         int                    `json:"failed"`
	Results        []MigrationCompanyItem `json:"results"`
	Status         string                 `json:"status"` // scheduled, in_progress, completed, failed
	ScheduledFor   *time.Time             `json:"scheduled_for,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Summary        string                 `json:"summary"`
}

// MigrationCompanyItem represents the result of migrating a single company
type MigrationCompanyItem struct {
	CompanyID      string `json:"company_id"`
	CompanyName    string `json:"company_name"`
	OrganizationID string `json:"organization_id"`
	Success        bool   `json:"success"`
	Error          string `json:"error,omitempty"`
}
