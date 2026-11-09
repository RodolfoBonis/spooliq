package usecases

import (
	"errors"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
)

// isUnauthorized reports whether err is an InfinitePay 401/403 (rejected key).
func isUnauthorized(err error) bool {
	return errors.Is(err, coreServices.ErrIPUnauthorized)
}

// isUnavailable reports whether err is an InfinitePay 5xx/timeout/transport error.
func isUnavailable(err error) bool {
	return errors.Is(err, coreServices.ErrIPUnavailable)
}
