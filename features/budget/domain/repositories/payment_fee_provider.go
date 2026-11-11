package repositories

import "context"

// PaymentFeeProvider supplies the payment fee rate (percent) an organization wants
// EMBEDDED into its budget prices. It is a narrow port implemented by the payments
// feature and injected via FX, so the budget package never imports payments (the
// dependency points one way, like PaymentDefaultsProvider).
//
// EmbeddedFeeRatePercent returns 0 whenever the fee must NOT be embedded: the
// integration is inactive (disconnected or disabled), the embed_fee_in_price switch is
// off, or the reference card rate (credit at assured_installments) is not configured.
// A rate of 0 therefore keeps a budget's price identical to a fee-less budget.
type PaymentFeeProvider interface {
	EmbeddedFeeRatePercent(ctx context.Context, organizationID string) (float64, error)
}
