package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentGatewayLinkModel represents the payment gateway integration for a company
// Relacionamento 1:1 com Company (uma company tem apenas um gateway)
// FK: OrganizationID → companies(organization_id) CASCADE (defined in CompanyModel side)
type PaymentGatewayLinkModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"organization_id"` // 1:1 - UNIQUE FK to companies
	Gateway        string         `gorm:"type:varchar(50);not null" json:"gateway"`                       // "asaas", "stripe", "paypal"
	CustomerID     string         `gorm:"type:varchar(255);not null;index" json:"customer_id"`            // ID do cliente no gateway
	SubscriptionID *string        `gorm:"type:varchar(255);index" json:"subscription_id"`                 // ID da subscription no gateway (nullable)
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Note: No reverse relationship to avoid circular import.
	// The FK constraint is defined on the CompanyModel side via PaymentGateway field.
}

// TableName specifies the table name for GORM
func (PaymentGatewayLinkModel) TableName() string {
	return "payment_gateway_links"
}

// BeforeCreate is a GORM hook executed before creating a payment gateway link
func (p *PaymentGatewayLinkModel) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (p *PaymentGatewayLinkModel) ToEntity() *entities.PaymentGatewayLinkEntity {
	return &entities.PaymentGatewayLinkEntity{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Gateway:        p.Gateway,
		CustomerID:     p.CustomerID,
		SubscriptionID: p.SubscriptionID,
		IsActive:       p.IsActive,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
		DeletedAt:      convertDeletedAt(p.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (p *PaymentGatewayLinkModel) FromEntity(entity *entities.PaymentGatewayLinkEntity) {
	p.ID = entity.ID
	p.OrganizationID = entity.OrganizationID
	p.Gateway = entity.Gateway
	p.CustomerID = entity.CustomerID
	p.SubscriptionID = entity.SubscriptionID
	p.IsActive = entity.IsActive
	p.CreatedAt = entity.CreatedAt
	p.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		p.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// convertDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func convertDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
