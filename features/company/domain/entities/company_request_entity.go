package entities

// CreateCompanyRequest represents the request to create a new company
type CreateCompanyRequest struct {
	Name      string  `json:"name" validate:"required,min=1,max=255"`
	TradeName *string `json:"trade_name,omitempty" validate:"omitempty,max=255"`
	Document  *string `json:"document,omitempty" validate:"omitempty,max=20"` // CNPJ
	Email     *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone     *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	WhatsApp  *string `json:"whatsapp,omitempty" validate:"omitempty,max=50"`
	Instagram *string `json:"instagram,omitempty" validate:"omitempty,max=255"`
	Website   *string `json:"website,omitempty" validate:"omitempty,url,max=255"`
	LogoURL   *string `json:"logo_url,omitempty" validate:"omitempty,url,max=500"`
	Address   *string `json:"address,omitempty" validate:"omitempty,max=500"`
	City      *string `json:"city,omitempty" validate:"omitempty,max=100"`
	State     *string `json:"state,omitempty" validate:"omitempty,max=100"`
	ZipCode   *string `json:"zip_code,omitempty" validate:"omitempty,max=20"`
}

// UpdateCompanyRequest represents the request to update an existing company
type UpdateCompanyRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	TradeName *string `json:"trade_name,omitempty" validate:"omitempty,max=255"`
	Document  *string `json:"document,omitempty" validate:"omitempty,max=20"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone     *string `json:"phone,omitempty" validate:"omitempty,max=50"`
	WhatsApp  *string `json:"whatsapp,omitempty" validate:"omitempty,max=50"`
	Instagram *string `json:"instagram,omitempty" validate:"omitempty,max=255"`
	Website   *string `json:"website,omitempty" validate:"omitempty,url,max=255"`
	LogoURL   *string `json:"logo_url,omitempty" validate:"omitempty,url,max=500"`
	Address   *string `json:"address,omitempty" validate:"omitempty,max=500"`
	City      *string `json:"city,omitempty" validate:"omitempty,max=100"`
	State     *string `json:"state,omitempty" validate:"omitempty,max=100"`
	ZipCode   *string `json:"zip_code,omitempty" validate:"omitempty,max=20"`
	// DefaultTaxRate is the organization's default "por dentro" tax rate (percent, 0..<100).
	DefaultTaxRate *float64 `json:"default_tax_rate,omitempty" validate:"omitempty,gte=0,lt=100"`
}
