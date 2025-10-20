package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionEntity represents a subscription payment history in the domain layer
type SubscriptionEntity struct {
	ID                  uuid.UUID   `json:"id"`
	OrganizationID      string      `json:"organization_id"`
	SubscriptionPlanID  *uuid.UUID  `json:"subscription_plan_id,omitempty"`  // FK to subscription_plans - which plan was paid
	PaymentMethodID     *uuid.UUID  `json:"payment_method_id,omitempty"`     // FK to payment_methods - which card was used
	AsaasPaymentID      string      `json:"asaas_payment_id,omitempty"`
	AsaasInvoiceID      string      `json:"asaas_invoice_id,omitempty"`
	AsaasCustomerID     string      `json:"asaas_customer_id,omitempty"`
	AsaasSubscriptionID string      `json:"asaas_subscription_id,omitempty"`
	Amount              float64     `json:"amount"`
	NetValue            float64     `json:"net_value"`
	Status              string      `json:"status"`
	BillingType         string      `json:"billing_type,omitempty"` // CREDIT_CARD, BOLETO, PIX
	EventType           string      `json:"event_type,omitempty"`   // Last webhook event received
	Description         string      `json:"description,omitempty"`
	PaymentDate         *time.Time  `json:"payment_date,omitempty"`
	DueDate             time.Time   `json:"due_date"`
	InvoiceURL          string      `json:"invoice_url,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	DeletedAt           *time.Time  `json:"deleted_at,omitempty"`
}

// Payment status constants (28 Payment events)
const (
	// Creation and Authorization
	StatusPending    = "pending"
	StatusAuthorized = "authorized"

	// Risk Analysis
	StatusAwaitingRiskAnalysis = "awaiting_risk_analysis"
	StatusApprovedByRisk       = "approved_by_risk"
	StatusReprovedByRisk       = "reproved_by_risk"

	// Confirmation and Receipt
	StatusConfirmed   = "confirmed"
	StatusReceived    = "received"
	StatusAnticipated = "anticipated"

	// Overdue and Updates
	StatusOverdue = "overdue"
	StatusUpdated = "updated"

	// Deletion and Restoration
	StatusDeleted  = "deleted"
	StatusRestored = "restored"

	// Refunds
	StatusRefunded          = "refunded"
	StatusPartiallyRefunded = "partially_refunded"
	StatusRefundInProgress  = "refund_in_progress"
	StatusRefundDenied      = "refund_denied"

	// Chargebacks
	StatusChargebackRequested        = "chargeback_requested"
	StatusChargebackDispute          = "chargeback_dispute"
	StatusAwaitingChargebackReversal = "awaiting_chargeback_reversal"

	// Dunning (Negativação)
	StatusDunningRequested = "dunning_requested"
	StatusDunningReceived  = "dunning_received"

	// Views (Analytics)
	StatusCheckoutViewed = "checkout_viewed"
	StatusBankSlipViewed = "bank_slip_viewed"

	// Special Operations
	StatusReceivedInCashUndone = "received_in_cash_undone"
	StatusCaptureRefused       = "capture_refused"

	// Split Operations
	StatusSplitCancelled = "split_cancelled"
	StatusSplitBlocked   = "split_blocked"
	StatusSplitUnblocked = "split_unblocked"

	// Legacy/Fallback
	StatusFailed = "failed"
)

// Subscription status constants (7 Subscription events)
const (
	SubscriptionCreated        = "subscription_created"
	SubscriptionUpdated        = "subscription_updated"
	SubscriptionInactivated    = "subscription_inactivated"
	SubscriptionDeleted        = "subscription_deleted"
	SubscriptionSplitDisabled  = "subscription_split_disabled"
	SubscriptionSplitBlocked   = "subscription_split_blocked"
	SubscriptionSplitUnblocked = "subscription_split_unblocked"
)
