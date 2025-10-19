package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentMethodModel represents the payment method data model for GORM
type PaymentMethodModel struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID       string         `gorm:"type:varchar(255);not null;index" json:"organization_id"`
	AsaasCreditCardToken string         `gorm:"type:varchar(255);not null" json:"asaas_credit_card_token"`
	HolderName           string         `gorm:"type:varchar(255);not null" json:"holder_name"`
	Last4Digits          string         `gorm:"type:varchar(4);not null" json:"last_4_digits"`
	Brand                string         `gorm:"type:varchar(50);not null" json:"brand"`
	ExpiryMonth          string         `gorm:"type:varchar(2);not null" json:"expiry_month"`
	ExpiryYear           string         `gorm:"type:varchar(4);not null" json:"expiry_year"`
	IsPrimary            bool           `gorm:"not null;default:false" json:"is_primary"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (PaymentMethodModel) TableName() string {
	return "payment_methods"
}

// BeforeCreate is a GORM hook executed before creating a payment method
func (p *PaymentMethodModel) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (p *PaymentMethodModel) ToEntity() *entities.PaymentMethodEntity {
	return &entities.PaymentMethodEntity{
		ID:                   p.ID,
		OrganizationID:       p.OrganizationID,
		AsaasCreditCardToken: p.AsaasCreditCardToken,
		HolderName:           p.HolderName,
		Last4Digits:          p.Last4Digits,
		Brand:                p.Brand,
		ExpiryMonth:          p.ExpiryMonth,
		ExpiryYear:           p.ExpiryYear,
		IsPrimary:            p.IsPrimary,
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
		DeletedAt:            getDeletedAt(p.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (p *PaymentMethodModel) FromEntity(entity *entities.PaymentMethodEntity) {
	p.ID = entity.ID
	p.OrganizationID = entity.OrganizationID
	p.AsaasCreditCardToken = entity.AsaasCreditCardToken
	p.HolderName = entity.HolderName
	p.Last4Digits = entity.Last4Digits
	p.Brand = entity.Brand
	p.ExpiryMonth = entity.ExpiryMonth
	p.ExpiryYear = entity.ExpiryYear
	p.IsPrimary = entity.IsPrimary
	p.CreatedAt = entity.CreatedAt
	p.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		p.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}
