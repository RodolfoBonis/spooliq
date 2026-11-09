package services

// InfinitePay Checkout Integrated API models.
//
// The Checkout API is public (no approval): each organization connects with its own
// API key, passed per call via the X-Api-Key header. Amounts are ALWAYS in cents.
//
// UNVERIFIED ASSUMPTIONS (no sandbox — confirm during the R$1 production test):
//   - POST /links accepts items[].price in cents, order_nsu, redirect_url,
//     customer{name,email,phone_number}, webhook{url,events:["payment"]},
//     options{single_payment, valid_until (RFC3339), assured_installments,
//     discounts.pix (integer %), payment_methods_enabled}, and an optional top-level
//     handle. Response: { checkout_url, slug? }.
//   - The webhook POSTs { handle, order_nsu, transaction_nsu, invoice_slug | slug }.
//     invoice_slug is the link slug; slug is a fallback.
//   - POST /payment_check accepts { handle, order_nsu, transaction_nsu?, slug? } and
//     returns at least { success, paid, paid_amount (cents), capture_method,
//     installments }. It MAY NOT echo transaction_nsu / receipt_url / order_nsu, so the
//     confirm path treats those as optional and falls back to the request values.

// InfinitePayItem is a line item in a checkout link. Price is in cents.
type InfinitePayItem struct {
	Name     string `json:"name,omitempty"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
}

// InfinitePayCustomer carries the (optional) customer identification shown at checkout.
type InfinitePayCustomer struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

// InfinitePayWebhook registers the confirmation callback for a link. The URL carries
// the per-integration secret token and must be masked in logs.
type InfinitePayWebhook struct {
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
}

// InfinitePayDiscounts configures per-method discounts. Pix is an integer percentage (0-30).
type InfinitePayDiscounts struct {
	Pix int `json:"pix,omitempty"`
}

// InfinitePayLinkOptions configures the link's payment behavior.
type InfinitePayLinkOptions struct {
	// AssuredInstallments is the number of interest-free installments the seller
	// absorbs (1-12).
	AssuredInstallments int `json:"assured_installments,omitempty"`
	// SinglePayment makes the link usable exactly once.
	SinglePayment bool `json:"single_payment"`
	// ValidUntil is the link expiry (RFC3339).
	ValidUntil string `json:"valid_until,omitempty"`
	// PaymentMethodsEnabled restricts the methods offered (e.g. "credit_card","pix").
	PaymentMethodsEnabled []string              `json:"payment_methods_enabled,omitempty"`
	Discounts             *InfinitePayDiscounts `json:"discounts,omitempty"`
}

// InfinitePayCreateLinkRequest is the body of POST /links.
type InfinitePayCreateLinkRequest struct {
	Handle      string                  `json:"handle,omitempty"`
	Items       []InfinitePayItem       `json:"items"`
	OrderNSU    string                  `json:"order_nsu"`
	RedirectURL string                  `json:"redirect_url"`
	Customer    *InfinitePayCustomer    `json:"customer,omitempty"`
	Webhook     *InfinitePayWebhook     `json:"webhook,omitempty"`
	Options     *InfinitePayLinkOptions `json:"options,omitempty"`
}

// InfinitePayCreateLinkResponse is the response of POST /links.
type InfinitePayCreateLinkResponse struct {
	CheckoutURL string `json:"checkout_url"`
	// Slug identifies the created link; echoed back on webhook/payment_check.
	Slug string `json:"slug,omitempty"`
}

// InfinitePayPaymentCheckRequest is the body of POST /payment_check. Any subset of
// identifiers may be provided; order_nsu alone is the minimum the confirmation path
// relies on.
type InfinitePayPaymentCheckRequest struct {
	Handle         string `json:"handle,omitempty"`
	OrderNSU       string `json:"order_nsu,omitempty"`
	TransactionNSU string `json:"transaction_nsu,omitempty"`
	Slug           string `json:"slug,omitempty"`
}

// InfinitePayPaymentCheckResponse is the response of POST /payment_check. Fields are
// modeled defensively: only success+paid+paid_amount are relied upon; the rest may be
// absent.
type InfinitePayPaymentCheckResponse struct {
	Success bool `json:"success"`
	Paid    bool `json:"paid"`
	// PaidAmount is the amount actually captured, in cents. It may exceed the charged
	// amount when the buyer pays card interest.
	PaidAmount     int64  `json:"paid_amount"`
	CaptureMethod  string `json:"capture_method"`
	Installments   int    `json:"installments"`
	OrderNSU       string `json:"order_nsu"`
	TransactionNSU string `json:"transaction_nsu"`
	Slug           string `json:"slug"`
	InvoiceSlug    string `json:"invoice_slug"`
	ReceiptURL     string `json:"receipt_url"`
}

// ResolvedSlug returns invoice_slug when present, else slug.
func (r *InfinitePayPaymentCheckResponse) ResolvedSlug() string {
	if r.InvoiceSlug != "" {
		return r.InvoiceSlug
	}
	return r.Slug
}

// InfinitePayErrorResponse is the error envelope returned by the Checkout API.
type InfinitePayErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
