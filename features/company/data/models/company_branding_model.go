package models

import (
	"time"

	"github.com/google/uuid"
)

// CompanyBrandingModel represents the database model for company branding configuration
type CompanyBrandingModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;unique;index" json:"organization_id"`
	TemplateName   string    `gorm:"type:varchar(50)" json:"template_name"` // "modern_pink", "corporate_blue", etc.

	// Header colors
	HeaderBgColor   string `gorm:"type:varchar(7);not null" json:"header_bg_color"`   // #HEX
	HeaderTextColor string `gorm:"type:varchar(7);not null" json:"header_text_color"` // #HEX

	// Primary colors
	PrimaryColor     string `gorm:"type:varchar(7);not null" json:"primary_color"`      // #HEX
	PrimaryTextColor string `gorm:"type:varchar(7);not null" json:"primary_text_color"` // #HEX

	// Secondary colors
	SecondaryColor     string `gorm:"type:varchar(7);not null" json:"secondary_color"`      // #HEX
	SecondaryTextColor string `gorm:"type:varchar(7);not null" json:"secondary_text_color"` // #HEX

	// Text colors
	TitleColor    string `gorm:"type:varchar(7);not null" json:"title_color"`     // #HEX
	BodyTextColor string `gorm:"type:varchar(7);not null" json:"body_text_color"` // #HEX

	// Accent colors
	AccentColor string `gorm:"type:varchar(7);not null" json:"accent_color"` // #HEX
	BorderColor string `gorm:"type:varchar(7);not null" json:"border_color"` // #HEX

	// Background colors
	BackgroundColor    string `gorm:"type:varchar(7);not null" json:"background_color"`       // #HEX
	TableHeaderBgColor string `gorm:"type:varchar(7);not null" json:"table_header_bg_color"`  // #HEX
	TableRowAltBgColor string `gorm:"type:varchar(7);not null" json:"table_row_alt_bg_color"` // #HEX

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (CompanyBrandingModel) TableName() string {
	return "company_branding"
}
