package entities

import (
	"time"

	"github.com/google/uuid"
)

// CompanyBrandingEntity represents the domain entity for company branding
type CompanyBrandingEntity struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID string    `json:"organization_id"`
	TemplateName   string    `json:"template_name"`

	// Header colors
	HeaderBgColor   string `json:"header_bg_color"`
	HeaderTextColor string `json:"header_text_color"`

	// Primary colors
	PrimaryColor     string `json:"primary_color"`
	PrimaryTextColor string `json:"primary_text_color"`

	// Secondary colors
	SecondaryColor     string `json:"secondary_color"`
	SecondaryTextColor string `json:"secondary_text_color"`

	// Text colors
	TitleColor    string `json:"title_color"`
	BodyTextColor string `json:"body_text_color"`

	// Accent colors
	AccentColor string `json:"accent_color"`
	BorderColor string `json:"border_color"`

	// Background colors
	BackgroundColor    string `json:"background_color"`
	TableHeaderBgColor string `json:"table_header_bg_color"`
	TableRowAltBgColor string `json:"table_row_alt_bg_color"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
