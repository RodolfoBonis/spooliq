package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentGatewayLinkEntity represents the payment gateway integration domain entity
type PaymentGatewayLinkEntity struct {
	ID             uuid.UUID
	OrganizationID string
	Gateway        string  // "asaas", "stripe", "paypal"
	CustomerID     string  // ID do cliente no gateway
	SubscriptionID *string // ID da subscription no gateway (nullable)
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}
