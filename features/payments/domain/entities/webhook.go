package entities

// WebhookPayload is the InfinitePay payment confirmation callback body. The webhook
// is unsigned, so these values are only HINTS: every state change is re-verified via
// payment_check with the organization's own credentials before being applied.
type WebhookPayload struct {
	Handle         string `json:"handle"`
	OrderNSU       string `json:"order_nsu"`
	TransactionNSU string `json:"transaction_nsu"`
	Slug           string `json:"slug"`
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
