package entities

// RegisterRequest represents the request to register a new company and owner user
type RegisterRequest struct {
	// User data
	Name     string `json:"name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`

	// Company data
	CompanyName      string `json:"company_name" validate:"required"`
	CompanyTradeName string `json:"company_trade_name"`
	CompanyDocument  string `json:"company_document" validate:"required"` // CNPJ
	CompanyPhone     string `json:"company_phone" validate:"required"`

	// Address
	Address       string `json:"address" validate:"required"`
	AddressNumber string `json:"address_number" validate:"required"`
	Complement    string `json:"complement"`
	Neighborhood  string `json:"neighborhood" validate:"required"`
	City          string `json:"city" validate:"required"`
	State         string `json:"state" validate:"required,len=2"`
	ZipCode       string `json:"zip_code" validate:"required"`
}

// RegisterResponse represents the response after successful registration
type RegisterResponse struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	TrialEndsAt    string `json:"trial_ends_at"`
	Message        string `json:"message"`
}
