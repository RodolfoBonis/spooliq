package entities

// WebhookPayload is the InfinitePay payment confirmation callback body. The webhook
// is unsigned, so these values are only HINTS: every state change is re-verified via
// payment_check with the organization's own credentials before being applied.
//
// Per the research the link slug arrives as `invoice_slug`; `slug` is kept as a
// fallback. ResolvedSlug() returns whichever is present.
type WebhookPayload struct {
	Handle         string `json:"handle"`
	OrderNSU       string `json:"order_nsu"`
	TransactionNSU string `json:"transaction_nsu"`
	InvoiceSlug    string `json:"invoice_slug"`
	Slug           string `json:"slug"`
}

// ResolvedSlug returns invoice_slug when present, else slug.
func (w WebhookPayload) ResolvedSlug() string {
	if w.InvoiceSlug != "" {
		return w.InvoiceSlug
	}
	return w.Slug
}

// ConfirmSource identifies who triggered a confirmation attempt, for logging and to
// drive subtly different behavior (e.g. the webhook may apply refunds, the redirect
// only confirms payments).
type ConfirmSource string

// Confirm sources.
const (
	SourceWebhook  ConfirmSource = "webhook"
	SourceRedirect ConfirmSource = "redirect"
	SourceJob      ConfirmSource = "job"
)
