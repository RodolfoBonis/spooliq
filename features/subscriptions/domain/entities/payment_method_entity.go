package entities

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethodEntity represents a payment method (credit card) in the domain layer
type PaymentMethodEntity struct {
	ID                   uuid.UUID  `json:"id"`
	OrganizationID       string     `json:"organization_id"`
	AsaasCreditCardToken string     `json:"asaas_credit_card_token"` // Token returned by Asaas
	HolderName           string     `json:"holder_name"`
	Last4Digits          string     `json:"last_4_digits"`
	Brand                string     `json:"brand"`                  // visa, mastercard, etc
	ExpiryMonth          string     `json:"expiry_month"`           // MM
	ExpiryYear           string     `json:"expiry_year"`            // YYYY
	IsPrimary            bool       `json:"is_primary"`             // Primary payment method
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"deleted_at,omitempty"`
}

// PaymentMethodCreateRequest represents the request to create/tokenize a payment method
type PaymentMethodCreateRequest struct {
	HolderName      string `json:"holder_name" binding:"required"`
	Number          string `json:"number" binding:"required"`          // Will be tokenized, not stored
	ExpiryMonth     string `json:"expiry_month" binding:"required"`
	ExpiryYear      string `json:"expiry_year" binding:"required"`
	Ccv             string `json:"ccv" binding:"required"`             // Will be sent to Asaas, not stored
	SetAsPrimary    bool   `json:"set_as_primary"`
}

// PaymentMethodResponse represents the response when fetching payment methods
type PaymentMethodResponse struct {
	ID             uuid.UUID `json:"id"`
	HolderName     string    `json:"holder_name"`
	Last4Digits    string    `json:"last_4_digits"`
	Brand          string    `json:"brand"`
	ExpiryMonth    string    `json:"expiry_month"`
	ExpiryYear     string    `json:"expiry_year"`
	IsPrimary      bool      `json:"is_primary"`
	CreatedAt      time.Time `json:"created_at"`
}
