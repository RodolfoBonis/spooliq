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

// openSupplyBackfillDB connects to TEST_DATABASE_URL in a throwaway schema with the
// supplies and budget_item_supplies tables shaped like production after the package
// columns were added by AutoMigrate (package_size default 1, package_price default
// 0). Skips when TEST_DATABASE_URL is unset.
func openSupplyBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping supply package backfill integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("supplypkg_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})
	for _, stmt := range []string{
		`CREATE TABLE supplies (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL, unit varchar(20) NOT NULL,
			package_size numeric NOT NULL DEFAULT 1, package_price bigint NOT NULL DEFAULT 0,
			unit_cost bigint NOT NULL DEFAULT 0)`,
		`CREATE TABLE budget_item_supplies (
			id uuid PRIMARY KEY, budget_item_id uuid NOT NULL, supply_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL, quantity_per_piece numeric NOT NULL DEFAULT 0,
			"order" integer NOT NULL DEFAULT 0, name varchar(255) NOT NULL, unit varchar(20) NOT NULL,
			package_size numeric NOT NULL DEFAULT 1, package_price bigint NOT NULL DEFAULT 0,
			unit_cost bigint NOT NULL DEFAULT 0)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

// TestSupplyPackageBackfillIdempotent verifies the package backfills seed
// package_price from the legacy unit_cost exactly once: a legacy row (unit_cost > 0,
// package_price 0) gets package_price = unit_cost, a free row (unit_cost 0) stays 0,
// and a row already carrying a package_price is never overwritten — and re-running
// both migrations changes nothing.
func TestSupplyPackageBackfillIdempotent(t *testing.T) {
	db := openSupplyBackfillDB(t)

	legacy, free, alreadyPackaged := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO supplies (id, organization_id, name, unit, package_size, package_price, unit_cost)
		VALUES (?, 'org', 'LED', 'un', 1, 0, 150),
		       (?, 'org', 'Free', 'un', 1, 0, 0),
		       (?, 'org', 'Lotion', 'ml', 400, 6990, 17)`, legacy, free, alreadyPackaged).Error)

	bisLegacy := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_item_supplies (id, budget_item_id, supply_id, organization_id, name, unit, package_size, package_price, unit_cost)
		VALUES (?, ?, ?, 'org', 'LED', 'un', 1, 0, 150)`, bisLegacy, uuid.New(), legacy).Error)

	supplyPkg := func(id uuid.UUID) (size float64, price, unit int64) {
		var row struct {
			PackageSize  float64
			PackagePrice int64
			UnitCost     int64
		}
		require.NoError(t, db.Raw(`SELECT package_size, package_price, unit_cost FROM supplies WHERE id = ?`, id).Scan(&row).Error)
		return row.PackageSize, row.PackagePrice, row.UnitCost
	}
	bisPrice := func(id uuid.UUID) int64 {
		var p int64
		require.NoError(t, db.Raw(`SELECT package_price FROM budget_item_supplies WHERE id = ?`, id).Scan(&p).Error)
		return p
	}

	// Run both backfills twice; idempotent guards must make the second run a no-op.
	for i := 0; i < 2; i++ {
		runDataMigration(t, db, "2026-10-30_supply_package_backfill")
		runDataMigration(t, db, "2026-10-30_budget_item_supply_package_backfill")
	}

	size, price, unit := supplyPkg(legacy)
	require.EqualValues(t, 1, size, "legacy package_size stays 1")
	require.EqualValues(t, 150, price, "legacy package_price seeded from unit_cost")
	require.EqualValues(t, 150, unit, "legacy unit_cost preserved")

	_, price, _ = supplyPkg(free)
	require.EqualValues(t, 0, price, "free supply (unit_cost 0) keeps package_price 0")

	size, price, _ = supplyPkg(alreadyPackaged)
	require.EqualValues(t, 400, size, "pre-packaged supply untouched")
	require.EqualValues(t, 6990, price, "pre-packaged package_price never overwritten")

	require.EqualValues(t, 150, bisPrice(bisLegacy), "snapshot package_price seeded from unit_cost")
}
