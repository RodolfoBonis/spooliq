package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProviderInfinitePay is the only supported provider for now.
const ProviderInfinitePay = "infinitepay"

// PaymentIntegration is the domain view of a per-organization payment connection.
// The ciphertext fields are internal and never serialized to clients.
type PaymentIntegration struct {
	ID                     uuid.UUID
	OrganizationID         string
	Provider               string
	Handle                 string
	APIKeyCiphertext       string
	APIKeyLast4            string
	WebhookTokenCiphertext string
	Enabled                bool
	AssuredInstallments    int
	PixDiscountPercent     int
	DefaultDepositPercent  *float64
	// FeeRates is the org's InfinitePay fee table (nil when not configured yet).
	FeeRates *FeeRates
	// EmbedFeeInPrice toggles embedding the reference card fee into the budget price.
	EmbedFeeInPrice bool
	VerifiedAt      *time.Time
	DisconnectedAt  *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Connected reports whether the integration still holds live credentials (not
// disconnected).
func (p *PaymentIntegration) Connected() bool {
	return p != nil && p.DisconnectedAt == nil && p.APIKeyCiphertext != ""
}

// Active reports whether the integration can currently originate charges: connected
// AND enabled.
func (p *PaymentIntegration) Active() bool {
	return p.Connected() && p.Enabled
}

// IntegrationResponse is the authenticated GET response. The API key is NEVER
// returned; only its last 4 characters. Sensitive fields (last4, pending_links) are
// omitted for the User role by the use case.
type IntegrationResponse struct {
	Provider              string   `json:"provider"`
	Connected             bool     `json:"connected"`
	Enabled               bool     `json:"enabled"`
	Handle                string   `json:"handle"`
	APIKeyLast4           string   `json:"api_key_last4,omitempty"`
	AssuredInstallments   int      `json:"assured_installments"`
	PixDiscountPercent    int      `json:"pix_discount_percent"`
	DefaultDepositPercent *float64 `json:"default_deposit_percent"`
	// FeeRates and EmbedFeeInPrice are non-sensitive settings included for every role.
	FeeRates        *FeeRates  `json:"fee_rates"`
	EmbedFeeInPrice bool       `json:"embed_fee_in_price"`
	VerifiedAt      *time.Time `json:"verified_at"`
	// PendingLinks counts open (pending) payment links across the org. Omitted for
	// the User role.
	PendingLinks *int64 `json:"pending_links,omitempty"`
}

// UpsertIntegrationRequest is the PUT body. APIKey is optional: when omitted the
// stored key is kept (only the other settings change). It is validated in the use
// case (Optional semantics are not needed; absent key == keep).
type UpsertIntegrationRequest struct {
	Handle                string   `json:"handle" validate:"required,max=255"`
	APIKey                *string  `json:"api_key" validate:"omitempty,min=8,max=255"`
	Enabled               bool     `json:"enabled"`
	AssuredInstallments   int      `json:"assured_installments" validate:"gte=1,lte=12"`
	PixDiscountPercent    int      `json:"pix_discount_percent" validate:"gte=0,lte=30"`
	DefaultDepositPercent *float64 `json:"default_deposit_percent" validate:"omitempty,gt=0,lt=100"`
	// FeeRates is the optional InfinitePay fee table, kept as raw JSON so the use case
	// can tell three cases apart (older desktops must not wipe the table):
	//   - key ABSENT        -> len == 0           -> keep the stored table;
	//   - explicit null      -> "null"            -> clear the table;
	//   - a JSON object      -> parse into FeeRates -> replace the table.
	FeeRates json.RawMessage `json:"fee_rates"`
	// EmbedFeeInPrice is a pointer so an ABSENT key keeps the stored flag; present
	// true/false sets it. It may only be true when the reference card rate exists.
	EmbedFeeInPrice *bool `json:"embed_fee_in_price"`
}
