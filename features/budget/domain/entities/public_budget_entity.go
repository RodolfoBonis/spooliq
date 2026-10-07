package entities

import "time"

// PublicBudgetItem is a sanitized budget item for the customer-facing public view.
// unit_price/total_price are the per-item SALE values (cents) already used on the
// PDF; no internal cost breakdown is exposed.
type PublicBudgetItem struct {
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"`
	ProductDimensions  *string `json:"product_dimensions,omitempty"`
	UnitPrice          int64   `json:"unit_price"`  // cents (sale, with markup)
	TotalPrice         int64   `json:"total_price"` // cents (sale, with markup)
}

// PublicBudgetCustomer exposes only the customer's name (no email/phone/document).
type PublicBudgetCustomer struct {
	Name string `json:"name"`
}

// PublicBudgetCompany is the subset of company fields shown to the customer.
type PublicBudgetCompany struct {
	Name      string  `json:"name"`
	TradeName *string `json:"trade_name,omitempty"`
	LogoURL   *string `json:"logo_url,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	WhatsApp  *string `json:"whatsapp,omitempty"`
	Instagram *string `json:"instagram,omitempty"`
	Website   *string `json:"website,omitempty"`
	City      *string `json:"city,omitempty"`
	State     *string `json:"state,omitempty"`
}

// PublicBudgetView is the SANITIZED budget representation returned by the public
// endpoints. It deliberately omits internal costs (filament/energy/labor/overhead/
// profit), organization_id, owner and the customer's email/phone/document.
type PublicBudgetView struct {
	QuoteNumber          *int       `json:"quote_number"`
	Name                 string     `json:"name"`
	Description          string     `json:"description"`
	Status               string     `json:"status"`
	ValidUntil           *time.Time `json:"valid_until"`
	IsExpired            bool       `json:"is_expired"`
	CanRespond           bool       `json:"can_respond"`
	CreatedAt            time.Time  `json:"created_at"`
	CustomerResponseAt   *time.Time `json:"customer_response_at"`
	CustomerResponseName *string    `json:"customer_response_name"`
	RejectionReason      *string    `json:"rejection_reason"`

	Customer PublicBudgetCustomer `json:"customer"`
	Company  PublicBudgetCompany  `json:"company"`
	Items    []PublicBudgetItem   `json:"items"`

	BasePrice      int64   `json:"base_price"`
	DiscountAmount int64   `json:"discount_amount"`
	ShippingCost   int64   `json:"shipping_cost"`
	TaxAmount      int64   `json:"tax_amount"`
	TaxRateApplied float64 `json:"tax_rate_applied"`
	Total          int64   `json:"total"`

	DeliveryDays *int    `json:"delivery_days"`
	PaymentTerms *string `json:"payment_terms"`
	Notes        *string `json:"notes"`
}
