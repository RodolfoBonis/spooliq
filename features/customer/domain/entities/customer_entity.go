package entities

import (
	"time"

	"github.com/google/uuid"
)

// CustomerEntity represents a customer in the domain layer
type CustomerEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"` // Multi-tenancy
	Name           string     `json:"name"`
	Email          *string    `json:"email,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	Document       *string    `json:"document,omitempty"` // CPF/CNPJ
	Address        *string    `json:"address,omitempty"`
	City           *string    `json:"city,omitempty"`
	State          *string    `json:"state,omitempty"`
	ZipCode        *string    `json:"zip_code,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	OwnerUserID    string     `json:"owner_user_id"` // For audit trail
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
