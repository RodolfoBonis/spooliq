package repositories_test

import (
	"fmt"
	"os"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/dashboard/data/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openDashboardTestDB creates an isolated schema (pinned to one connection so the
// search_path persists) with the minimal tables the low-stock widget reads. Gated by
// TEST_DATABASE_URL.
func openDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping dashboard low-stock integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	schema := fmt.Sprintf("dash_ls_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	for _, s := range []string{
		`CREATE TABLE brands (id uuid PRIMARY KEY, organization_id varchar(255), name varchar(255), deleted_at timestamptz)`,
		`CREATE TABLE materials (id uuid PRIMARY KEY, organization_id varchar(255), name varchar(255), deleted_at timestamptz)`,
		`CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL DEFAULT '',
			color varchar(100) NOT NULL DEFAULT '',
			color_hex varchar(7),
			brand_id uuid,
			material_id uuid,
			stock_grams bigint NOT NULL DEFAULT 0,
			track_stock boolean NOT NULL DEFAULT false,
			low_stock_threshold_grams integer,
			deleted_at timestamptz
		)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func dashInsertFilament(t *testing.T, db *gorm.DB, org, name string, stock int64, track bool, threshold *int) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO filaments (id, organization_id, name, color, stock_grams, track_stock, low_stock_threshold_grams) VALUES (?,?,?,?,?,?,?)`,
		uuid.New(), org, name, "red", stock, track, threshold,
	).Error)
}

// TestIntegration_GetLowStockFilaments proves the widget returns only tracked,
// under-threshold filaments, ordered by shortfall ascending, and is org-scoped.
func TestIntegration_GetLowStockFilaments(t *testing.T) {
	db := openDashboardTestDB(t)
	repo := repoimpl.NewDashboardRepository(db)

	const orgA, orgB = "dash-org-a", "dash-org-b"
	th := func(v int) *int { return &v }

	dashInsertFilament(t, db, orgA, "verylow", 10, true, th(100))     // shortfall -90
	dashInsertFilament(t, db, orgA, "slightlylow", 90, true, th(100)) // shortfall -10
	dashInsertFilament(t, db, orgA, "healthy", 500, true, th(100))    // excluded
	dashInsertFilament(t, db, orgA, "untracked", 0, false, th(100))   // excluded
	dashInsertFilament(t, db, orgA, "nothresh", 0, true, nil)         // excluded
	dashInsertFilament(t, db, orgB, "other-org", 0, true, th(100))    // excluded by org

	rows, err := repo.GetLowStockFilaments(orgA, 20)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "verylow", rows[0].Name, "most depleted first")
	require.Equal(t, "slightlylow", rows[1].Name)
	require.Equal(t, int64(10), rows[0].StockGrams)
	require.Equal(t, 100, rows[0].LowStockThresholdGrams)
}
