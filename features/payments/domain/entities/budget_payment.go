package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentRecordStatus is the lifecycle status of a single budget payment.
type PaymentRecordStatus string

// Payment record statuses.
const (
	PaymentPending  PaymentRecordStatus = "pending"
	PaymentPaid     PaymentRecordStatus = "paid"
	PaymentRefunded PaymentRecordStatus = "refunded"
	PaymentCanceled PaymentRecordStatus = "canceled"
	PaymentFailed   PaymentRecordStatus = "failed"
)

// status_reason values recorded on a payment when its status changes.
const (
	ReasonSuperseded      = "superseded"        // a new link with a different amount replaced this one
	ReasonBudgetCancelled = "budget_cancelled"  // the budget was cancelled
	ReasonBudgetRevised   = "budget_revised"    // the budget went back to draft for a revision
	ReasonExpired         = "expired"           // the link passed its valid_until
	ReasonDisconnected    = "disconnected"      // the integration was disconnected
	ReasonPaidAfterCancel = "paid_after_cancel" // a cancelled link was paid anyway (manual refund needed)
)

// BudgetPayment is the domain view of one payment attempt against a budget.
type BudgetPayment struct {
	ID              uuid.UUID
	OrganizationID  string
	BudgetID        uuid.UUID
	Kind            ChargeKind
	OrderNSU        string
	TransactionNSU  *string
	Slug            *string
	AmountCents     int64
	PaidAmountCents int64
	Installments    int
	CaptureMethod   string
	CheckoutURL     string
	ReceiptURL      *string
	Status          PaymentRecordStatus
	StatusReason    string
	LinkExpiresAt   *time.Time
	PaidAt          *time.Time
	CanceledAt      *time.Time
	RefundedAt      *time.Time
	LastCheckedAt   *time.Time
	CheckAttempts   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PaymentLineResponse is one payment row in the authenticated budget payments list.
// It exposes non-sensitive fields only (no nsu, no slug). status_reason is included
// so the owner UI can show the "reembolse pela InfinitePay" note for paid_after_cancel;
// it is intentionally NOT exposed on the public payment line.
type PaymentLineResponse struct {
	Kind          ChargeKind `json:"kind"`
	Amount        int64      `json:"amount"`
	PaidAmount    int64      `json:"paid_amount"`
	CaptureMethod string     `json:"capture_method"`
	Installments  int        `json:"installments"`
	Status        string     `json:"status"`
	StatusReason  string     `json:"status_reason,omitempty"`
	CheckoutURL   string     `json:"checkout_url,omitempty"`
	ReceiptURL    *string    `json:"receipt_url"`
	PaidAt        *time.Time `json:"paid_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// BudgetPaymentsResponse is the authenticated GET /budgets/:id/payments response.
type BudgetPaymentsResponse struct {
	Status         PaymentStatus         `json:"status"`
	Total          int64                 `json:"total"`
	Paid           int64                 `json:"paid"`
	Balance        int64                 `json:"balance"`
	DepositPercent *float64              `json:"deposit_percent"`
	Payments       []PaymentLineResponse `json:"payments"`
}

// CreateLinkResponse is the authenticated POST /budgets/:id/payments/link response.
type CreateLinkResponse struct {
	CheckoutURL string     `json:"checkout_url"`
	Kind        ChargeKind `json:"kind"`
	Amount      int64      `json:"amount"`
	ExpiresAt   *time.Time `json:"expires_at"`
}
