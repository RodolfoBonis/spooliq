package entities

import "time"

// Unavailable reasons for the public payment block's can_pay=false states.
const (
	UnavailableNotApproved  = "not_approved"      // budget is sent, not yet approved
	UnavailablePaymentsOff  = "payments_disabled" // integration disabled
	UnavailableAlreadyPaid  = "already_paid"      // nothing left to charge
	UnavailableBelowMinimum = "below_minimum"     // remaining amount under R$ 1
	UnavailableNotPayable   = "not_payable"       // budget in a non-payable status
)

// PublicPaymentNext describes the next charge the customer can make, with the Pix
// and installment hints shown on the page. Interest values themselves are shown on
// the InfinitePay screen (there is no simulation API).
type PublicPaymentNext struct {
	Kind                     ChargeKind `json:"kind"`
	Amount                   int64      `json:"amount"`
	PixAmount                int64      `json:"pix_amount"`
	PixDiscountPercent       float64    `json:"pix_discount_percent"`
	InterestFreeInstallments int        `json:"interest_free_installments"`
	InstallmentAmount        int64      `json:"installment_amount"`
	MaxInstallments          int        `json:"max_installments"`
}

// PublicPaymentLine is a sanitized paid-payment row for the public view. It never
// exposes internal identifiers (nsu, slug).
type PublicPaymentLine struct {
	Kind          ChargeKind `json:"kind"`
	Amount        int64      `json:"amount"`
	PaidAmount    int64      `json:"paid_amount"`
	CaptureMethod string     `json:"capture_method"`
	Installments  int        `json:"installments"`
	PaidAt        *time.Time `json:"paid_at"`
	ReceiptURL    *string    `json:"receipt_url"`
}

// PublicPaymentBlock is the "payment" object embedded in the public budget view.
// It is intentionally self-contained so the web/desktop clients render the whole
// payment UI from it.
type PublicPaymentBlock struct {
	Enabled           bool                `json:"enabled"`
	Status            PaymentStatus       `json:"status"`
	Total             int64               `json:"total"`
	Paid              int64               `json:"paid"`
	Balance           int64               `json:"balance"`
	DepositPercent    *float64            `json:"deposit_percent"`
	CanPay            bool                `json:"can_pay"`
	UnavailableReason string              `json:"unavailable_reason,omitempty"`
	Next              *PublicPaymentNext  `json:"next"`
	HasPendingLink    bool                `json:"has_pending_link"`
	Payments          []PublicPaymentLine `json:"payments"`
}

// CreatePublicPaymentRequest is the POST /public/budgets/:token/payments body.
type CreatePublicPaymentRequest struct {
	Kind ChargeKind `json:"kind"`
}

// ConfirmPublicPaymentRequest is the POST /public/budgets/:token/payments/confirm body.
type ConfirmPublicPaymentRequest struct {
	OrderNSU       string `json:"order_nsu"`
	TransactionNSU string `json:"transaction_nsu"`
	Slug           string `json:"slug"`
	ReceiptURL     string `json:"receipt_url"`
}

// ConfirmState is the "confirmation" field on the public confirm response.
type ConfirmState string

// Confirmation states.
const (
	ConfirmPaidState       ConfirmState = "paid"
	ConfirmProcessingState ConfirmState = "processing"
)
