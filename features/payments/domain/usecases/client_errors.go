package usecases

import (
	"errors"

	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	"github.com/jackc/pgx/v5/pgconn"
)

// isUnauthorized reports whether err is an InfinitePay 401/403 (rejected key).
func isUnauthorized(err error) bool {
	return errors.Is(err, coreServices.ErrIPUnauthorized)
}

// isUnavailable reports whether err is an InfinitePay 5xx/timeout/transport error.
func isUnavailable(err error) bool {
	return errors.Is(err, coreServices.ErrIPUnavailable)
}

// isUniqueViolation reports whether err is a PostgreSQL unique-violation (23505),
// e.g. a reused transaction_nsu hitting the partial unique index.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
