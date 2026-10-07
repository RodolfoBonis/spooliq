package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestIntegration_UpdateItem_PersistsZeroCosts exercises the REAL GORM repository
// against a real PostgreSQL to prove the BLOCKER fix: UpdateItem must persist
// zero-valued cost columns. The previous Updates(struct) skipped zero fields, so
// turning energy off left a stale energy_cost on the item even though the budget
// totals (recomputed from scratch) dropped to 0.
//
// It is gated by TEST_DATABASE_URL and skips when unset. To run locally:
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/budget/data/repositories/...
//
// The test creates an isolated schema and a minimal budget_items table via raw
// SQL (no foreign keys, no association graph) so it is fully self-contained and
// drops everything on cleanup.
func TestIntegration_UpdateItem_PersistsZeroCosts(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping budget item UpdateItem integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// Pin to a single connection so SET search_path persists for the whole test.
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("budget_it_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	// Minimal budget_items table carrying only the columns UpdateItem touches
	// plus the primary key and tenant scope. No FKs required.
	require.NoError(t, db.Exec(`
		CREATE TABLE budget_items (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			filament_cost bigint NOT NULL DEFAULT 0,
			waste_cost bigint NOT NULL DEFAULT 0,
			energy_cost bigint NOT NULL DEFAULT 0,
			machine_cost bigint NOT NULL DEFAULT 0,
			setup_cost bigint NOT NULL DEFAULT 0,
			manual_labor_cost bigint NOT NULL DEFAULT 0,
			post_processing_cost bigint NOT NULL DEFAULT 0,
			support_removal_cost bigint NOT NULL DEFAULT 0,
			packaging_cost bigint NOT NULL DEFAULT 0,
			quality_control_cost bigint NOT NULL DEFAULT 0,
			failure_cost bigint NOT NULL DEFAULT 0,
			item_total_cost bigint NOT NULL DEFAULT 0,
			unit_price bigint NOT NULL DEFAULT 0,
			updated_at timestamptz
		)
	`).Error)

	const org = "org-a"
	itemID := uuid.New()

	// Seed an item that already has non-zero costs (as if energy was enabled).
	require.NoError(t, db.Exec(`
		INSERT INTO budget_items
			(id, organization_id, filament_cost, waste_cost, energy_cost, setup_cost, manual_labor_cost, item_total_cost, unit_price, updated_at)
		VALUES (?, ?, 1200, 300, 500, 200, 100, 2300, 2300, ?)
	`, itemID, org, time.Now()).Error)

	repo := repoimpl.NewBudgetRepository(db)

	// Recalculation turned energy (and waste/labor) off: every cost is now zero.
	zeroItem := &entities.BudgetItemEntity{
		ID:              itemID,
		OrganizationID:  org,
		FilamentCost:    0,
		WasteCost:       0,
		EnergyCost:      0,
		SetupCost:       0,
		ManualLaborCost: 0,
		ItemTotalCost:   0,
		UnitPrice:       0,
	}
	require.NoError(t, repo.UpdateItem(context.Background(), zeroItem))

	var got struct {
		FilamentCost    int64
		WasteCost       int64
		EnergyCost      int64
		SetupCost       int64
		ManualLaborCost int64
		ItemTotalCost   int64
		UnitPrice       int64
	}
	require.NoError(t, db.Raw(`
		SELECT filament_cost, waste_cost, energy_cost, setup_cost, manual_labor_cost, item_total_cost, unit_price
		FROM budget_items WHERE id = ?
	`, itemID).Scan(&got).Error)

	require.Equal(t, int64(0), got.EnergyCost, "energy_cost must be persisted as 0 after energy is turned off")
	require.Equal(t, int64(0), got.FilamentCost)
	require.Equal(t, int64(0), got.WasteCost)
	require.Equal(t, int64(0), got.SetupCost)
	require.Equal(t, int64(0), got.ManualLaborCost)
	require.Equal(t, int64(0), got.ItemTotalCost)
	require.Equal(t, int64(0), got.UnitPrice)

	// Cross-tenant guard: an UpdateItem scoped to another org must not touch the row.
	require.NoError(t, db.Exec(`UPDATE budget_items SET energy_cost = 777 WHERE id = ?`, itemID).Error)
	otherOrgItem := *zeroItem
	otherOrgItem.OrganizationID = "org-b"
	require.NoError(t, repo.UpdateItem(context.Background(), &otherOrgItem))

	var energyAfter int64
	require.NoError(t, db.Raw(`SELECT energy_cost FROM budget_items WHERE id = ?`, itemID).Scan(&energyAfter).Error)
	require.Equal(t, int64(777), energyAfter, "UpdateItem from another org must not modify the row")
}
