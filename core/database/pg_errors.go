// Package database holds small, dependency-aware helpers for working with the
// SQL driver, kept separate from business logic. It currently exposes typed
// detection of the PostgreSQL error codes that callers map to HTTP responses.
package database

import (
	stderrors "errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL SQLSTATE codes we classify. See
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	sqlStateForeignKeyViolation = "23503"
	sqlStateUniqueViolation     = "23505"
)

// IsForeignKeyViolation reports whether err (anywhere in its wrap chain) is a
// PostgreSQL foreign key constraint violation (SQLSTATE 23503). Handlers use it
// to turn a RESTRICT delete into a clean 409 instead of a 500. It inspects the
// typed *pgconn.PgError rather than matching on message text.
func IsForeignKeyViolation(err error) bool {
	return hasPgCode(err, sqlStateForeignKeyViolation)
}

// IsUniqueViolation reports whether err is a PostgreSQL unique constraint
// violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	return hasPgCode(err, sqlStateUniqueViolation)
}

func hasPgCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}
