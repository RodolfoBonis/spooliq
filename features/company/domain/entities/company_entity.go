package entities

import (
	"time"

	"github.com/google/uuid"
)

// CompanyEntity represents a company in the domain layer
type CompanyEntity struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Name           string     `json:"name"`
	TradeName      *string    `json:"trade_name,omitempty"`
	Document       *string    `json:"document,omitempty"` // CNPJ
	Email          *string    `json:"email,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	WhatsApp       *string    `json:"whatsapp,omitempty"`
	Instagram      *string    `json:"instagram,omitempty"`
	Website        *string    `json:"website,omitempty"`
	LogoURL        *string    `json:"logo_url,omitempty"`
	Address        *string    `json:"address,omitempty"`
	City           *string    `json:"city,omitempty"`
	State          *string    `json:"state,omitempty"`
	ZipCode        *string    `json:"zip_code,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
