package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
)

// SubscriptionModel represents the subscription payment history data model for GORM
type SubscriptionModel struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID      string     `gorm:"type:varchar(255);not null;index"`
	AsaasPaymentID      string     `gorm:"type:varchar(255);index"`
	AsaasInvoiceID      string     `gorm:"type:varchar(255)"`
	AsaasCustomerID     string     `gorm:"type:varchar(255)"`
	AsaasSubscriptionID string     `gorm:"type:varchar(255)"`
	Amount              float64    `gorm:"type:decimal(10,2)"`
	NetValue            float64    `gorm:"type:decimal(10,2)"`
	Status              string     `gorm:"type:varchar(50);index"` // 35 possible statuses
	BillingType         string     `gorm:"type:varchar(20)"`       // CREDIT_CARD, BOLETO, PIX
	EventType           string     `gorm:"type:varchar(100)"`      // Last webhook event type
	Description         string     `gorm:"type:text"`
	PaymentDate         *time.Time `gorm:"type:timestamp"`
	DueDate             time.Time  `gorm:"type:timestamp"`
	InvoiceURL          string     `gorm:"type:text"`
	CreatedAt           time.Time  `gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (SubscriptionModel) TableName() string {
	return "subscription_payments"
}

// ToEntity converts the GORM model to domain entity
func (s *SubscriptionModel) ToEntity() *entities.SubscriptionEntity {
	return &entities.SubscriptionEntity{
		ID:                  s.ID,
		OrganizationID:      s.OrganizationID,
		AsaasPaymentID:      s.AsaasPaymentID,
		AsaasInvoiceID:      s.AsaasInvoiceID,
		AsaasCustomerID:     s.AsaasCustomerID,
		AsaasSubscriptionID: s.AsaasSubscriptionID,
		Amount:              s.Amount,
		NetValue:            s.NetValue,
		Status:              s.Status,
		BillingType:         s.BillingType,
		EventType:           s.EventType,
		Description:         s.Description,
		PaymentDate:         s.PaymentDate,
		DueDate:             s.DueDate,
		InvoiceURL:          s.InvoiceURL,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (s *SubscriptionModel) FromEntity(entity *entities.SubscriptionEntity) {
	s.ID = entity.ID
	s.OrganizationID = entity.OrganizationID
	s.AsaasPaymentID = entity.AsaasPaymentID
	s.AsaasInvoiceID = entity.AsaasInvoiceID
	s.AsaasCustomerID = entity.AsaasCustomerID
	s.AsaasSubscriptionID = entity.AsaasSubscriptionID
	s.Amount = entity.Amount
	s.NetValue = entity.NetValue
	s.Status = entity.Status
	s.BillingType = entity.BillingType
	s.EventType = entity.EventType
	s.Description = entity.Description
	s.PaymentDate = entity.PaymentDate
	s.DueDate = entity.DueDate
	s.InvoiceURL = entity.InvoiceURL
	s.CreatedAt = entity.CreatedAt
	s.UpdatedAt = entity.UpdatedAt
}
