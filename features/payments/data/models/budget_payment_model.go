package models

import (
	"encoding/json"
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/google/uuid"
)

// BudgetPaymentModel is one payment attempt (link) against a budget. A budget can
// have at most one PENDING payment per kind, enforced by a partial unique index on
// (budget_id, kind) WHERE status = 'pending' (added in RunMigrations). order_nsu is
// globally unique; transaction_nsu is unique when present (partial unique index).
type BudgetPaymentModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_budget_payments_org" json:"organization_id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index:idx_budget_payments_budget" json:"budget_id"`

	// Identification
	Kind           string  `gorm:"type:varchar(10);not null" json:"kind"`
	OrderNSU       string  `gorm:"type:varchar(64);not null" json:"order_nsu"`
	TransactionNSU *string `gorm:"type:varchar(64)" json:"transaction_nsu"`
	Slug           *string `gorm:"type:varchar(128)" json:"slug"`

	// Values (cents)
	AmountCents     int64  `gorm:"type:bigint;not null" json:"amount_cents"`
	PaidAmountCents int64  `gorm:"type:bigint;not null;default:0" json:"paid_amount_cents"`
	Installments    int    `gorm:"type:integer;not null;default:0" json:"installments"`
	CaptureMethod   string `gorm:"type:varchar(20)" json:"capture_method"`

	// Links
	CheckoutURL string  `gorm:"type:text" json:"checkout_url"`
	ReceiptURL  *string `gorm:"type:text" json:"receipt_url"`

	// Lifecycle
	Status       string `gorm:"type:varchar(12);not null;default:'pending'" json:"status"`
	StatusReason string `gorm:"type:varchar(32)" json:"status_reason"`

	LinkExpiresAt *time.Time `gorm:"type:timestamptz" json:"link_expires_at"`
	PaidAt        *time.Time `gorm:"type:timestamptz" json:"paid_at"`
	CanceledAt    *time.Time `gorm:"type:timestamptz" json:"canceled_at"`
	RefundedAt    *time.Time `gorm:"type:timestamptz" json:"refunded_at"`

	// Control
	LastCheckedAt *time.Time      `gorm:"type:timestamptz" json:"last_checked_at"`
	CheckAttempts int             `gorm:"type:integer;not null;default:0" json:"check_attempts"`
	RawEvent      json.RawMessage `gorm:"type:jsonb" json:"raw_event,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_budget_payments_status_created" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (BudgetPaymentModel) TableName() string {
	return "budget_payments"
}

// ToEntity maps the model to its domain entity.
func (m *BudgetPaymentModel) ToEntity() *entities.BudgetPayment {
	return &entities.BudgetPayment{
		ID:              m.ID,
		OrganizationID:  m.OrganizationID,
		BudgetID:        m.BudgetID,
		Kind:            entities.ChargeKind(m.Kind),
		OrderNSU:        m.OrderNSU,
		TransactionNSU:  m.TransactionNSU,
		Slug:            m.Slug,
		AmountCents:     m.AmountCents,
		PaidAmountCents: m.PaidAmountCents,
		Installments:    m.Installments,
		CaptureMethod:   m.CaptureMethod,
		CheckoutURL:     m.CheckoutURL,
		ReceiptURL:      m.ReceiptURL,
		Status:          entities.PaymentRecordStatus(m.Status),
		StatusReason:    m.StatusReason,
		LinkExpiresAt:   m.LinkExpiresAt,
		PaidAt:          m.PaidAt,
		CanceledAt:      m.CanceledAt,
		RefundedAt:      m.RefundedAt,
		LastCheckedAt:   m.LastCheckedAt,
		CheckAttempts:   m.CheckAttempts,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
