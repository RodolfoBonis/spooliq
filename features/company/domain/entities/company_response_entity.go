package entities

import (
	"time"
)

// CompanyResponse represents the response structure for a company
type CompanyResponse struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	TradeName      *string   `json:"trade_name,omitempty"`
	Document       *string   `json:"document,omitempty"`
	Email          *string   `json:"email,omitempty"`
	Phone          *string   `json:"phone,omitempty"`
	WhatsApp       *string   `json:"whatsapp,omitempty"`
	Instagram      *string   `json:"instagram,omitempty"`
	Website        *string   `json:"website,omitempty"`
	LogoURL        *string   `json:"logo_url,omitempty"`
	Address        *string   `json:"address,omitempty"`
	City           *string   `json:"city,omitempty"`
	State          *string   `json:"state,omitempty"`
	ZipCode        *string   `json:"zip_code,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
