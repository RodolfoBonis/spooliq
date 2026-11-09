package services

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openBackfillDB connects to TEST_DATABASE_URL in a throwaway schema (skips when unset).
func openBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping lifecycle backfill integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("lifecycle_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})
	for _, stmt := range []string{
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL,
			status varchar(20) NOT NULL, valid_until timestamptz,
			customer_response_at timestamptz, customer_response_name varchar(120),
			approved_at timestamptz, completed_at timestamptz,
			updated_at timestamptz NOT NULL DEFAULT now())`,
		`CREATE TABLE budget_status_history (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(), budget_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL, previous_status varchar(20) NOT NULL,
			new_status varchar(20) NOT NULL, changed_by varchar(255) NOT NULL,
			notes text, created_at timestamptz NOT NULL DEFAULT now())`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func runDataMigration(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	for _, m := range dataMigrations {
		if m.name == name {
			require.NoError(t, db.Exec(m.sql).Error, name)
			return
		}
	}
	t.Fatalf("data migration %s not found", name)
}

func TestLifecycleBackfills(t *testing.T) {
	db := openBackfillDB(t)
	day := func(n int) time.Time { return time.Date(2026, 9, n, 12, 0, 0, 0, time.UTC) }

	publicApproved, publicRejected, manualCompleted, expired, reopened :=
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ins := func(id uuid.UUID, status string, response, validUntil *time.Time, updated time.Time) {
		require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, status, customer_response_at, customer_response_name, valid_until, updated_at)
			VALUES (?, 'org', ?, ?, 'Ana', ?, ?)`, id, status, response, validUntil, updated).Error)
	}
	d2, d3, d5 := day(2), day(3), day(5)
	ins(publicApproved, "printing", &d2, nil, day(4))
	ins(publicRejected, "rejected", &d3, nil, d3)
	ins(manualCompleted, "completed", nil, nil, day(9))
	ins(expired, "expired", nil, &d5, day(6))
	ins(reopened, "draft", &d2, nil, day(7)) // responded then reopened: no history inferred
	// manualCompleted already has history written by the API.
	require.NoError(t, db.Exec(`INSERT INTO budget_status_history (budget_id, organization_id, previous_status, new_status, changed_by, created_at)
		VALUES (?, 'org', 'sent', 'approved', 'u1', ?), (?, 'org', 'printing', 'completed', 'u1', ?)`,
		manualCompleted, day(7), manualCompleted, day(8)).Error)

	names := []string{
		"2026-10-09_backfill_public_response_history",
		"2026-10-09_backfill_expired_history",
		"2026-10-09_backfill_budget_approved_at",
		"2026-10-09_backfill_budget_completed_at",
	}
	for _, n := range names {
		runDataMigration(t, db, n)
	}
	// Re-running must not duplicate history (guards are idempotent).
	runDataMigration(t, db, names[0])
	runDataMigration(t, db, names[1])

	countHistory := func(id uuid.UUID, status string) int64 {
		var c int64
		require.NoError(t, db.Raw(`SELECT count(*) FROM budget_status_history WHERE budget_id = ? AND new_status = ?`, id, status).Scan(&c).Error)
		return c
	}
	require.EqualValues(t, 1, countHistory(publicApproved, "approved"))
	require.EqualValues(t, 1, countHistory(publicRejected, "rejected"))
	require.EqualValues(t, 1, countHistory(expired, "expired"))
	require.EqualValues(t, 0, countHistory(reopened, "approved"))

	dates := func(id uuid.UUID) (approved, completed *time.Time) {
		var row struct{ ApprovedAt, CompletedAt *time.Time }
		require.NoError(t, db.Raw(`SELECT approved_at, completed_at FROM budgets WHERE id = ?`, id).Scan(&row).Error)
		return row.ApprovedAt, row.CompletedAt
	}
	a, c := dates(publicApproved)
	require.True(t, a.Equal(d2), "approval date comes from the customer response")
	require.Nil(t, c)
	a, c = dates(manualCompleted)
	require.True(t, a.Equal(day(7)))
	require.True(t, c.Equal(day(8)))
	a, _ = dates(publicRejected)
	require.Nil(t, a)
}
