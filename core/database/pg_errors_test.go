package database

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsForeignKeyViolation(t *testing.T) {
	fk := &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}
	if !IsForeignKeyViolation(fk) {
		t.Error("expected FK violation to be detected")
	}
	// Wrapped error is still detected via errors.As.
	if !IsForeignKeyViolation(fmt.Errorf("repo: %w", fk)) {
		t.Error("expected wrapped FK violation to be detected")
	}
	// Unique violation is not a FK violation.
	if IsForeignKeyViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("unique violation must not be classified as FK violation")
	}
	// Non-pg errors and nil return false.
	if IsForeignKeyViolation(fmt.Errorf("some other error")) {
		t.Error("plain error must not be classified as FK violation")
	}
	if IsForeignKeyViolation(nil) {
		t.Error("nil must not be classified as FK violation")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Error("expected unique violation to be detected")
	}
	if IsUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Error("FK violation must not be classified as unique violation")
	}
	if IsUniqueViolation(nil) {
		t.Error("nil must not be classified as unique violation")
	}
}
