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
	ReasonSuperseded      = "superseded"        // a new link with a different amount/settings replaced this one
	ReasonBudgetCancelled = "budget_cancelled"  // the budget was cancelled
	ReasonBudgetRevised   = "budget_revised"    // the budget went back to draft for a revision
	ReasonExpired         = "expired"           // the link passed its valid_until + grace
	ReasonDisconnected    = "disconnected"      // the integration was disconnected
	ReasonPaidAfterCancel = "paid_after_cancel" // a cancelled link was paid anyway (manual refund needed)
	ReasonAccountChanged  = "account_changed"   // the org reconnected with a different key/handle
)

// review_reason values when a payment is flagged for manual review.
const (
	ReviewValueDivergence = "value_divergence"
	ReviewOrderMismatch   = "order_mismatch"
	ReviewSlugMismatch    = "slug_mismatch"
	ReviewUnauthorized    = "unauthorized"
	ReviewNoCredentials   = "no_credentials"
	ReviewPossibleRefund  = "possible_refund"
)

// cancelReasonsRequiringRefund is the set of cancellation reasons that make a later
// payment a "paid_after_cancel" (money received on a link that should not have been
// payable). A link canceled because it EXPIRED is NOT in this set: InfinitePay
// enforces valid_until itself, so a paid expired link was paid in time and settles
// normally.
var cancelReasonsRequiringRefund = map[string]bool{
	ReasonBudgetCancelled: true,
	ReasonBudgetRevised:   true,
	ReasonDisconnected:    true,
	ReasonSuperseded:      true,
	ReasonAccountChanged:  true,
}

// IsRefundReason reports whether a cancellation reason makes a subsequent payment a
// paid_after_cancel (manual-refund) case.
func IsRefundReason(reason string) bool {
	return cancelReasonsRequiringRefund[reason]
}

// BudgetPayment is the domain view of one payment attempt against a budget.
type BudgetPayment struct {
	ID                 uuid.UUID
	OrganizationID     string
	BudgetID           uuid.UUID
	Kind               ChargeKind
	OrderNSU           string
	TransactionNSU     *string
	Slug               *string
	Handle             string
	AmountCents        int64
	PaidAmountCents    int64
	Installments       int
	CaptureMethod      string
	PixDiscountPercent int
	CheckoutURL        string
	ReceiptURL         *string
	LinkCreatingAt     *time.Time
	Status             PaymentRecordStatus
	StatusReason       string
	NeedsReview        bool
	ReviewReason       string
	LinkExpiresAt      *time.Time
	PaidAt             *time.Time
	CanceledAt         *time.Time
	RefundedAt         *time.Time
	LastCheckedAt      *time.Time
	CheckAttempts      int
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
	NeedsReview   bool       `json:"needs_review,omitempty"`
	CheckoutURL   string     `json:"checkout_url,omitempty"`
	ReceiptURL    *string    `json:"receipt_url"`
	PaidAt        *time.Time `json:"paid_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// BudgetPaymentsResponse is the authenticated GET /budgets/:id/payments response.
// RefundDue is the sum of paid_after_cancel principal (money received on a cancelled
// link that must be refunded via InfinitePay); it is NOT part of Paid.
type BudgetPaymentsResponse struct {
	Status         PaymentStatus         `json:"status"`
	Total          int64                 `json:"total"`
	Paid           int64                 `json:"paid"`
	Balance        int64                 `json:"balance"`
	RefundDue      int64                 `json:"refund_due"`
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
