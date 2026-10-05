package repositories

import (
	"errors"

	profileEntities "github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// uniqueViolationCode is the PostgreSQL SQLSTATE for a unique_violation.
const uniqueViolationCode = "23505"

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505), checking both GORM's translated error and the raw
// pgconn error so it works regardless of gorm.Config{TranslateError}.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}

// translateDefaultErr maps a partial-unique-index violation (a losing race the
// advisory lock did not cover) to the domain ErrDefaultConflict so callers
// return HTTP 409 with a friendly message instead of a raw DB error / 500.
func translateDefaultErr(err error) error {
	if isUniqueViolation(err) {
		return profileEntities.ErrDefaultConflict
	}
	return err
}

// profileDefaultLockKey builds the advisory-lock key serializing default
// mutations for an organization's profiles.
func profileDefaultLockKey(organizationID string) string {
	return "profile:" + organizationID
}

// lockDefaults takes a transaction-scoped advisory lock so concurrent default
// mutations for the same organization serialize instead of racing on the
// partial unique index. Released automatically at transaction end.
func lockDefaults(tx *gorm.DB, organizationID string) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", profileDefaultLockKey(organizationID)).Error
}
