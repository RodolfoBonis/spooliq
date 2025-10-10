package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// SubscriptionModel represents the subscription payment history data model for GORM
type SubscriptionModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID string     `gorm:"type:varchar(255);not null;index"`
	AsaasPaymentID string     `gorm:"type:varchar(255)"`
	AsaasInvoiceID string     `gorm:"type:varchar(255)"`
	Amount         float64    `gorm:"type:decimal(10,2)"`
	Status         string     `gorm:"type:varchar(20)"` // pending, confirmed, received, overdue, failed
	PaymentDate    *time.Time `gorm:"type:timestamp"`
	DueDate        time.Time  `gorm:"type:timestamp"`
	InvoiceURL     string     `gorm:"type:text"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
}

// TableName specifies the table name for GORM
func (SubscriptionModel) TableName() string {
	return "subscription_payments"
}

// ToEntity converts the GORM model to domain entity
func (s *SubscriptionModel) ToEntity() *entities.SubscriptionEntity {
	return &entities.SubscriptionEntity{
		ID:             s.ID,
		OrganizationID: s.OrganizationID,
		AsaasPaymentID: s.AsaasPaymentID,
		AsaasInvoiceID: s.AsaasInvoiceID,
		Amount:         s.Amount,
		Status:         s.Status,
		PaymentDate:    s.PaymentDate,
		DueDate:        s.DueDate,
		InvoiceURL:     s.InvoiceURL,
		CreatedAt:      s.CreatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (s *SubscriptionModel) FromEntity(entity *entities.SubscriptionEntity) {
	s.ID = entity.ID
	s.OrganizationID = entity.OrganizationID
	s.AsaasPaymentID = entity.AsaasPaymentID
	s.AsaasInvoiceID = entity.AsaasInvoiceID
	s.Amount = entity.Amount
	s.Status = entity.Status
	s.PaymentDate = entity.PaymentDate
	s.DueDate = entity.DueDate
	s.InvoiceURL = entity.InvoiceURL
	s.CreatedAt = entity.CreatedAt
}
