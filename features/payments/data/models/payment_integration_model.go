package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/google/uuid"
)

// PaymentIntegrationModel is the per-organization payment provider connection
// (currently only InfinitePay). It is 1:1 per (organization_id, provider), enforced
// by a unique index added in RunMigrations. Credentials are stored encrypted; only
// the last 4 characters of the API key are kept in clear for display.
type PaymentIntegrationModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_payment_integrations_org" json:"organization_id"`
	Provider       string    `gorm:"type:varchar(20);not null;default:'infinitepay'" json:"provider"`

	Handle                 string `gorm:"type:varchar(255)" json:"handle"`
	APIKeyCiphertext       string `gorm:"type:text" json:"-"`
	APIKeyLast4            string `gorm:"type:varchar(4)" json:"api_key_last4"`
	WebhookTokenCiphertext string `gorm:"type:text" json:"-"`

	Enabled               bool     `gorm:"not null;default:false" json:"enabled"`
	AssuredInstallments   int      `gorm:"not null;default:1" json:"assured_installments"`
	PixDiscountPercent    float64  `gorm:"type:double precision;not null;default:0" json:"pix_discount_percent"`
	DefaultDepositPercent *float64 `gorm:"type:double precision" json:"default_deposit_percent"` // null = total

	VerifiedAt     *time.Time `gorm:"type:timestamptz" json:"verified_at"`
	DisconnectedAt *time.Time `gorm:"type:timestamptz" json:"disconnected_at"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM.
func (PaymentIntegrationModel) TableName() string {
	return "payment_integrations"
}

// ToEntity maps the model to its domain entity. The ciphertexts are carried so the
// use cases can decrypt them; they are never serialized to JSON.
func (m *PaymentIntegrationModel) ToEntity() *entities.PaymentIntegration {
	return &entities.PaymentIntegration{
		ID:                     m.ID,
		OrganizationID:         m.OrganizationID,
		Provider:               m.Provider,
		Handle:                 m.Handle,
		APIKeyCiphertext:       m.APIKeyCiphertext,
		APIKeyLast4:            m.APIKeyLast4,
		WebhookTokenCiphertext: m.WebhookTokenCiphertext,
		Enabled:                m.Enabled,
		AssuredInstallments:    m.AssuredInstallments,
		PixDiscountPercent:     m.PixDiscountPercent,
		DefaultDepositPercent:  m.DefaultDepositPercent,
		VerifiedAt:             m.VerifiedAt,
		DisconnectedAt:         m.DisconnectedAt,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
}
