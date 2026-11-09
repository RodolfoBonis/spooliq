package usecases

import (
	"errors"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
)

// Sentinel errors returned by the payment services. Handlers map them to the public
// HTTP contract via MapChargeError / MapError.
var (
	// ErrPaymentsNotConfigured: no encryption key (or provider base URL) is set.
	ErrPaymentsNotConfigured = errors.New("payments_not_configured")
	// ErrPaymentsDisabled: the integration is disconnected/disabled or the global
	// kill switch is off.
	ErrPaymentsDisabled = errors.New("payments_disabled")
	// ErrChargeNotAvailable: the budget is not in a payable state (or no charge due).
	ErrChargeNotAvailable = errors.New("payment_not_available")
	// ErrChargeKindMismatch: the requested kind does not match the next due charge.
	ErrChargeKindMismatch = errors.New("payment_kind_mismatch")
	// ErrAlreadyPaid: the budget is fully paid.
	ErrAlreadyPaid = errors.New("budget_already_paid")
	// ErrBelowMinimum: the next charge is under R$ 1.
	ErrBelowMinimum = errors.New("payment_amount_below_minimum")
	// ErrProviderUnavailable: InfinitePay returned 5xx / timed out.
	ErrProviderUnavailable = errors.New("payment_provider_unavailable")
	// ErrInvalidCredentials: InfinitePay rejected the API key (401/403).
	ErrInvalidCredentials = errors.New("infinitepay_invalid_credentials")
	// ErrProviderRejected: InfinitePay rejected a well-formed request (4xx).
	ErrProviderRejected = errors.New("infinitepay_request_rejected")
)

// MapChargeError maps a charge/confirm sentinel error to the public HTTP envelope.
// Unknown errors fall through to a generic 500.
func MapChargeError(err error) *coreErrors.APIError {
	switch {
	case errors.Is(err, ErrPaymentsNotConfigured):
		return coreErrors.ServiceUnavailable("payments_not_configured", "Pagamentos não estão configurados no momento.")
	case errors.Is(err, ErrPaymentsDisabled):
		return coreErrors.Conflict("payments_disabled", "Pagamentos não estão disponíveis para este orçamento.")
	case errors.Is(err, ErrChargeNotAvailable):
		return coreErrors.Conflict("payment_not_available", "Pagamento não disponível para este orçamento.")
	case errors.Is(err, ErrChargeKindMismatch):
		return coreErrors.Conflict("payment_kind_mismatch", "A cobrança solicitada não corresponde ao valor devido.")
	case errors.Is(err, ErrAlreadyPaid):
		return coreErrors.Conflict("budget_already_paid", "Este orçamento já está pago.")
	case errors.Is(err, ErrBelowMinimum):
		return coreErrors.Unprocessable("payment_amount_below_minimum", "O valor a pagar é menor que o mínimo de R$ 1,00.")
	case errors.Is(err, ErrProviderUnavailable):
		return coreErrors.ServiceUnavailable("payment_provider_unavailable", "O provedor de pagamento está indisponível. Tente novamente em instantes.")
	case errors.Is(err, ErrInvalidCredentials):
		return coreErrors.Unprocessable("infinitepay_invalid_credentials", "As credenciais da InfinitePay foram recusadas.")
	case errors.Is(err, ErrProviderRejected):
		return coreErrors.Unprocessable("infinitepay_request_rejected", "O provedor de pagamento recusou a solicitação.")
	default:
		return coreErrors.Internal()
	}
}
