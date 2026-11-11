package models

import (
	"encoding/json"
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/google/uuid"
)

// BudgetPaymentModel is one payment attempt (link) against a budget. A budget can
// have at most one PENDING payment per kind, enforced by a partial unique index on
// (budget_id, kind) WHERE status = 'pending' (added in RunPaymentsMigrations).
// order_nsu is globally unique; transaction_nsu is unique when present (partial
// unique index).
type BudgetPaymentModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_budget_payments_org" json:"organization_id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index:idx_budget_payments_budget" json:"budget_id"`

	// Identification
	Kind           string  `gorm:"type:varchar(10);not null" json:"kind"`
	OrderNSU       string  `gorm:"type:varchar(64);not null" json:"order_nsu"`
	TransactionNSU *string `gorm:"type:varchar(64)" json:"transaction_nsu"`
	Slug           *string `gorm:"type:varchar(128)" json:"slug"`
	// Handle is the InfinitePay handle SNAPSHOT at link creation, so payment_check can
	// address the correct account even if the org later reconnects with another handle.
	Handle string `gorm:"type:varchar(255)" json:"handle"`

	// Values (cents)
	AmountCents     int64  `gorm:"type:bigint;not null" json:"amount_cents"`
	PaidAmountCents int64  `gorm:"type:bigint;not null;default:0" json:"paid_amount_cents"`
	Installments    int    `gorm:"type:integer;not null;default:0" json:"installments"`
	CaptureMethod   string `gorm:"type:varchar(20)" json:"capture_method"`
	// PixDiscountPercent is the Pix discount (integer %) SNAPSHOT at link creation, so
	// the confirm floor and the link's Pix price are stable even if settings change.
	PixDiscountPercent int `gorm:"type:integer;not null;default:0" json:"pix_discount_percent"`

	// Fee SNAPSHOT at confirmation: the InfinitePay fee rate (percent) and amount
	// (cents) computed from the org's fee table when the payment was marked paid. Both
	// are NULL when no fee table was configured at confirm time; the GET then estimates
	// from the current table.
	FeeRatePercent *float64 `gorm:"type:double precision" json:"fee_rate_percent"`
	FeeCents       *int64   `gorm:"type:bigint" json:"fee_cents"`

	// Links
	CheckoutURL string  `gorm:"type:text" json:"checkout_url"`
	ReceiptURL  *string `gorm:"type:text" json:"receipt_url"`
	// LinkCreatingAt is a short lease held while the remote CreateLink call is in
	// flight, so two concurrent requests never call the gateway twice for one order.
	LinkCreatingAt *time.Time `gorm:"type:timestamptz" json:"link_creating_at"`

	// Lifecycle
	Status       string `gorm:"type:varchar(12);not null;default:'pending'" json:"status"`
	StatusReason string `gorm:"type:varchar(32);not null;default:''" json:"status_reason"`

	// NeedsReview flags a row that requires manual attention (value divergence, order
	// mismatch, unauthorized, no credentials). The reconcile job skips these rows.
	NeedsReview  bool   `gorm:"not null;default:false" json:"needs_review"`
	ReviewReason string `gorm:"type:varchar(40);not null;default:''" json:"review_reason"`

	LinkExpiresAt *time.Time `gorm:"type:timestamptz" json:"link_expires_at"`
	PaidAt        *time.Time `gorm:"type:timestamptz" json:"paid_at"`
	CanceledAt    *time.Time `gorm:"type:timestamptz" json:"canceled_at"`
	RefundedAt    *time.Time `gorm:"type:timestamptz" json:"refunded_at"`

	// Control. The composite (status, created_at) index is created in
	// RunPaymentsMigrations (no GORM index tag here so the raw index is not shadowed).
	LastCheckedAt *time.Time      `gorm:"type:timestamptz" json:"last_checked_at"`
	CheckAttempts int             `gorm:"type:integer;not null;default:0" json:"check_attempts"`
	RawEvent      json.RawMessage `gorm:"type:jsonb" json:"raw_event,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (BudgetPaymentModel) TableName() string {
	return "budget_payments"
}

// ToEntity maps the model to its domain entity.
func (m *BudgetPaymentModel) ToEntity() *entities.BudgetPayment {
	return &entities.BudgetPayment{
		ID:                 m.ID,
		OrganizationID:     m.OrganizationID,
		BudgetID:           m.BudgetID,
		Kind:               entities.ChargeKind(m.Kind),
		OrderNSU:           m.OrderNSU,
		TransactionNSU:     m.TransactionNSU,
		Slug:               m.Slug,
		Handle:             m.Handle,
		AmountCents:        m.AmountCents,
		PaidAmountCents:    m.PaidAmountCents,
		Installments:       m.Installments,
		CaptureMethod:      m.CaptureMethod,
		PixDiscountPercent: m.PixDiscountPercent,
		FeeRatePercent:     m.FeeRatePercent,
		FeeCents:           m.FeeCents,
		CheckoutURL:        m.CheckoutURL,
		ReceiptURL:         m.ReceiptURL,
		LinkCreatingAt:     m.LinkCreatingAt,
		Status:             entities.PaymentRecordStatus(m.Status),
		StatusReason:       m.StatusReason,
		NeedsReview:        m.NeedsReview,
		ReviewReason:       m.ReviewReason,
		LinkExpiresAt:      m.LinkExpiresAt,
		PaidAt:             m.PaidAt,
		CanceledAt:         m.CanceledAt,
		RefundedAt:         m.RefundedAt,
		LastCheckedAt:      m.LastCheckedAt,
		CheckAttempts:      m.CheckAttempts,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}
