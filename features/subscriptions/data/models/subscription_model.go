package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionModel represents the subscription payment history data model for GORM
// FK: OrganizationID → companies(organization_id) RESTRICT (defined in CompanyModel side)
// FK: SubscriptionPlanID → subscription_plans(id) SET NULL
// FK: PaymentMethodID → payment_methods(id) SET NULL
type SubscriptionModel struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrganizationID      string         `gorm:"type:varchar(255);not null;index"` // FK to companies
	SubscriptionPlanID  *uuid.UUID     `gorm:"type:uuid;index"`                  // FK to subscription_plans - qual plano foi pago
	PaymentMethodID     *uuid.UUID     `gorm:"type:uuid;index"`                  // FK to payment_methods - qual cartão foi usado
	AsaasPaymentID      string         `gorm:"type:varchar(255);index"`
	AsaasInvoiceID      string         `gorm:"type:varchar(255)"`
	AsaasCustomerID     string         `gorm:"type:varchar(255)"`
	AsaasSubscriptionID string         `gorm:"type:varchar(255)"`
	Amount              float64        `gorm:"type:decimal(10,2)"`
	NetValue            float64        `gorm:"type:decimal(10,2)"`
	Status              string         `gorm:"type:varchar(50);index"` // 35 possible statuses
	BillingType         string         `gorm:"type:varchar(20)"`       // CREDIT_CARD, BOLETO, PIX
	EventType           string         `gorm:"type:varchar(100)"`      // Last webhook event type
	Description         string         `gorm:"type:text"`
	PaymentDate         *time.Time     `gorm:"type:timestamp"`
	DueDate             time.Time      `gorm:"type:timestamp"`
	InvoiceURL          string         `gorm:"type:text"`
	CreatedAt           time.Time      `gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// GORM v2 Relationships
	// Note: No Organization relationship to avoid circular import. FK defined in CompanyModel.
	Plan          *SubscriptionPlanModel `gorm:"foreignKey:SubscriptionPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"plan,omitempty"`
	PaymentMethod *PaymentMethodModel    `gorm:"foreignKey:PaymentMethodID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"payment_method,omitempty"`
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
		SubscriptionPlanID:  s.SubscriptionPlanID,
		PaymentMethodID:     s.PaymentMethodID,
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
		DeletedAt:           getDeletedAt(s.DeletedAt),
	}
}

// FromEntity converts domain entity to GORM model
func (s *SubscriptionModel) FromEntity(entity *entities.SubscriptionEntity) {
	s.ID = entity.ID
	s.OrganizationID = entity.OrganizationID
	s.SubscriptionPlanID = entity.SubscriptionPlanID
	s.PaymentMethodID = entity.PaymentMethodID
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
	if entity.DeletedAt != nil {
		s.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}
