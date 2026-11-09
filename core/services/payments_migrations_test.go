package services

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// TestRunPaymentsMigrations verifies the payments migration creates the tables and
// the critical unique indexes on a fresh schema.
func TestRunPaymentsMigrations(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payments migration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("paymig_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)

	// Minimal parent tables the FK constraints reference.
	require.NoError(t, db.Exec(`CREATE TABLE companies (organization_id varchar(255) PRIMARY KEY)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE budgets (id uuid PRIMARY KEY)`).Error)

	// Point the global connector at this schema for the duration of the test.
	prev := Connector
	Connector = db
	t.Cleanup(func() {
		Connector = prev
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	require.NotPanics(t, func() { RunPaymentsMigrations() })

	// Tables exist.
	for _, table := range []string{"payment_integrations", "budget_payments", "payment_webhook_events"} {
		var exists bool
		require.NoError(t, db.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?)", table).Scan(&exists).Error)
		require.True(t, exists, "table %s should exist", table)
	}

	// Critical + composite indexes exist.
	for _, idx := range []string{
		"uq_payment_integrations_org_provider",
		"uq_budget_payments_order_nsu",
		"uq_budget_payments_transaction_nsu",
		"uq_budget_payments_pending",
		"idx_budget_payments_status_created",
	} {
		var exists bool
		require.NoError(t, db.Raw("SELECT EXISTS(SELECT FROM pg_indexes WHERE schemaname = current_schema() AND indexname = ?)", idx).Scan(&exists).Error)
		require.True(t, exists, "index %s should exist", idx)
	}

	// Re-running is idempotent.
	require.NotPanics(t, func() { RunPaymentsMigrations() })
}
