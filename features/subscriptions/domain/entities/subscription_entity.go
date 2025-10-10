package entities

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionEntity represents a subscription payment history in the domain layer
type SubscriptionEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"`
	AsaasPaymentID string     `json:"asaas_payment_id,omitempty"`
	AsaasInvoiceID string     `json:"asaas_invoice_id,omitempty"`
	Amount         float64    `json:"amount"`
	Status         string     `json:"status"` // pending, confirmed, received, overdue, failed
	PaymentDate    *time.Time `json:"payment_date,omitempty"`
	DueDate        time.Time  `json:"due_date"`
	InvoiceURL     string     `json:"invoice_url,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Subscription status constants
const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusReceived  = "received"
	StatusOverdue   = "overdue"
	StatusFailed    = "failed"
)
