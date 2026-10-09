package repositories_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/stock/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/entities"
	stockrepos "github.com/RodolfoBonis/spooliq/features/stock/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// These tests exercise the REAL GORM stock repository against PostgreSQL. Gated by
// TEST_DATABASE_URL:
//
//	make infrastructure/raise
//	TEST_DATABASE_URL='postgres://user:password@localhost:5432/spooliq_db?sslmode=disable' \
//	    go test ./features/stock/...

const (
	orgA = "stock-org-a"
	orgB = "stock-org-b"
)

// openStockTestDB creates an isolated schema and returns a gorm.DB whose connection
// pool pins search_path to that schema via the libpq "options" parameter, so even the
// concurrency test (which needs multiple connections) sees the same tables. The schema
// is dropped on cleanup.
func openStockTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping stock integration test")
	}

	schema := fmt.Sprintf("stock_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))

	// Bootstrap connection to create the schema.
	boot, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")
	require.NoError(t, boot.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)

	// Working connection with search_path pinned for every pooled connection.
	db, err := gorm.Open(postgres.Open(withSearchPath(t, dsn, schema)), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = boot.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
		if sqlDB, e := boot.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})

	createStockSchema(t, db)
	return db
}

// withSearchPath appends the libpq "options" parameter so new connections default to
// the test schema. It supports the URL DSN form used by TEST_DATABASE_URL.
func withSearchPath(t *testing.T, dsn, schema string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	require.NoError(t, err, "TEST_DATABASE_URL must be a URL DSN")
	q := u.Query()
	q.Set("options", fmt.Sprintf("-c search_path=%s", schema))
	u.RawQuery = q.Encode()
	return u.String()
}

// createStockSchema creates the minimal tables the stock paths touch. No FKs needed.
func createStockSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL DEFAULT '',
			color varchar(100) NOT NULL DEFAULT '',
			stock_grams bigint NOT NULL DEFAULT 0,
			track_stock boolean NOT NULL DEFAULT false,
			low_stock_threshold_grams integer,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now(),
			deleted_at timestamptz
		)`,
		`CREATE TABLE cost_presets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			waste_grams_per_color_change numeric NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			quote_number integer,
			cost_preset_id uuid
		)`,
		`CREATE TABLE budget_items (
			id uuid PRIMARY KEY,
			budget_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			cost_preset_id uuid
		)`,
		`CREATE TABLE budget_item_filaments (
			id uuid PRIMARY KEY,
			budget_item_id uuid NOT NULL,
			filament_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL
		)`,
		`CREATE TABLE filament_stock_movements (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			organization_id varchar(255) NOT NULL,
			filament_id uuid NOT NULL,
			type varchar(20) NOT NULL,
			grams bigint NOT NULL,
			unit_price_per_kg bigint,
			budget_id uuid,
			note varchar(500),
			created_by varchar(255) NOT NULL,
			created_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE UNIQUE INDEX uq_mv_budget_filament_consumption ON filament_stock_movements(budget_id, filament_id) WHERE type = 'consumption'`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
}

func insertFilament(t *testing.T, db *gorm.DB, org string, stock int64, track bool, threshold *int) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO filaments (id, organization_id, name, color, stock_grams, track_stock, low_stock_threshold_grams) VALUES (?,?,?,?,?,?,?)`,
		id, org, "PLA", "red", stock, track, threshold,
	).Error)
	return id
}

func stockOf(t *testing.T, db *gorm.DB, id uuid.UUID) int64 {
	t.Helper()
	var s int64
	require.NoError(t, db.Raw(`SELECT stock_grams FROM filaments WHERE id = ?`, id).Scan(&s).Error)
	return s
}

// TestIntegration_ConcurrentMovements_CorrectFinalStock proves the FOR UPDATE row lock
// serializes concurrent movements so no increment is lost.
func TestIntegration_ConcurrentMovements_CorrectFinalStock(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	fil := insertFilament(t, db, orgA, 0, false, nil)

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := repo.CreateManualMovement(context.Background(), &entities.StockMovementEntity{
				OrganizationID: orgA, FilamentID: fil, Type: entities.MovementPurchase, Grams: 10, CreatedBy: "u1",
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	require.Equal(t, int64(n*10), stockOf(t, db, fil), "all concurrent increments must land")
}

// TestIntegration_Purchase_EnablesTracking proves any manual movement turns track_stock on.
func TestIntegration_Purchase_EnablesTracking(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	fil := insertFilament(t, db, orgA, 0, false, nil)

	_, summary, err := repo.CreateManualMovement(context.Background(), &entities.StockMovementEntity{
		OrganizationID: orgA, FilamentID: fil, Type: entities.MovementPurchase, Grams: 500, CreatedBy: "u1",
	})
	require.NoError(t, err)
	require.True(t, summary.TrackStock)
	require.Equal(t, int64(500), summary.StockGrams)

	var track bool
	require.NoError(t, db.Raw(`SELECT track_stock FROM filaments WHERE id = ?`, fil).Scan(&track).Error)
	require.True(t, track)
}

// TestIntegration_NegativeStockAllowed proves a movement may drive stock negative.
func TestIntegration_NegativeStockAllowed(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	fil := insertFilament(t, db, orgA, 100, true, nil)

	_, summary, err := repo.CreateManualMovement(context.Background(), &entities.StockMovementEntity{
		OrganizationID: orgA, FilamentID: fil, Type: entities.MovementAdjustment, Grams: -500, CreatedBy: "u1",
	})
	require.NoError(t, err)
	require.Equal(t, int64(-400), summary.StockGrams)
}

// TestIntegration_OrgIsolation_Create404 proves a movement for another org's filament
// is rejected (record not found -> 404 at the use case).
func TestIntegration_OrgIsolation_Create404(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	fil := insertFilament(t, db, orgA, 0, false, nil)

	_, _, err := repo.CreateManualMovement(context.Background(), &entities.StockMovementEntity{
		OrganizationID: orgB, FilamentID: fil, Type: entities.MovementPurchase, Grams: 10, CreatedBy: "u1",
	})
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestIntegration_Deduct_OnceAndIdempotent_SkipsUntracked proves completion deducts
// tracked filaments exactly once, skips untracked ones, and a retry never double-counts.
func TestIntegration_Deduct_OnceAndIdempotent_SkipsUntracked(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)

	tracked := insertFilament(t, db, orgA, 1000, true, nil)
	untracked := insertFilament(t, db, orgA, 1000, false, nil)

	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, quote_number) VALUES (?,?,?)`, budgetID, orgA, 7).Error)
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, organization_id) VALUES (?,?,?)`, item, budgetID, orgA).Error)
	// One item with THREE filament rows (no cost preset => default 15g/change). N=3 => 2
	// color changes => 30g purge, split equally = 10g per row. "tracked" occupies two rows
	// so it takes 800g quantity + 20g purge = 820g; "untracked" is skipped.
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, tracked, orgA, 500.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, tracked, orgA, 300.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, untracked, orgA, 300.0).Error)

	deduct := func() {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return repo.DeductForCompletedBudget(context.Background(), tx, budgetID, orgA, "sys")
		}))
	}

	deduct()
	require.Equal(t, int64(180), stockOf(t, db, tracked), "tracked decremented by 820 (800 + 20g purge) once")
	require.Equal(t, int64(1000), stockOf(t, db, untracked), "untracked skipped")

	// The consumption note records the purge grams included in the deduction.
	var note *string
	require.NoError(t, db.Raw(`SELECT note FROM filament_stock_movements WHERE budget_id = ? AND filament_id = ? AND type = 'consumption'`, budgetID, tracked).Scan(&note).Error)
	require.NotNil(t, note)
	require.Equal(t, "Consumo do orçamento (inclui 20 g de purga)", *note)

	// Retry must not double-deduct.
	deduct()
	require.Equal(t, int64(180), stockOf(t, db, tracked), "retry must not double-deduct")

	var consumptionCount int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM filament_stock_movements WHERE budget_id = ? AND type = 'consumption'`, budgetID).Scan(&consumptionCount).Error)
	require.Equal(t, int64(1), consumptionCount, "exactly one consumption movement for the tracked filament")
}

// TestIntegration_ListMovements_FilterAndQuoteNumber proves the ledger lists org-scoped
// movements, filters by type and resolves the linked budget's quote number.
func TestIntegration_ListMovements_FilterAndQuoteNumber(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	fil := insertFilament(t, db, orgA, 0, false, nil)

	ctx := context.Background()
	_, _, err := repo.CreateManualMovement(ctx, &entities.StockMovementEntity{OrganizationID: orgA, FilamentID: fil, Type: entities.MovementPurchase, Grams: 100, CreatedBy: "u1"})
	require.NoError(t, err)
	// waste is stored already-signed by the use case; the repo persists what it is given.
	_, _, err = repo.CreateManualMovement(ctx, &entities.StockMovementEntity{OrganizationID: orgA, FilamentID: fil, Type: entities.MovementWaste, Grams: -50, CreatedBy: "u1"})
	require.NoError(t, err)

	// Consumption movement tied to a budget with a quote number.
	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, quote_number) VALUES (?,?,?)`, budgetID, orgA, 42).Error)
	require.NoError(t, db.Exec(`INSERT INTO filament_stock_movements (id, organization_id, filament_id, type, grams, budget_id, created_by) VALUES (gen_random_uuid(),?,?, 'consumption', ?, ?, 'sys')`, orgA, fil, -70, budgetID).Error)

	all, total, err := repo.ListMovements(ctx, fil, orgA, "", "m.created_at desc, m.id desc", 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)

	waste, wtotal, err := repo.ListMovements(ctx, fil, orgA, "waste", "m.created_at desc, m.id desc", 20, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), wtotal)
	require.Len(t, waste, 1)
	require.Equal(t, int64(-50), waste[0].Grams)

	// The consumption row carries the budget quote number.
	var foundQuote *int
	for _, m := range all {
		if m.Type == entities.MovementConsumption {
			foundQuote = m.BudgetQuoteNumber
		}
	}
	require.NotNil(t, foundQuote)
	require.Equal(t, 42, *foundQuote)
}

// deductItemWithPresets builds a one-item budget with two tracked filaments (100g each)
// and the given item/budget cost preset IDs, deducts it, and returns the two filaments.
func deductItemWithPresets(t *testing.T, db *gorm.DB, repo stockrepos.StockRepository, itemPreset, budgetPreset *uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	a := insertFilament(t, db, orgA, 1000, true, nil)
	b := insertFilament(t, db, orgA, 1000, true, nil)

	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, quote_number, cost_preset_id) VALUES (?,?,?,?)`, budgetID, orgA, 1, budgetPreset).Error)
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, organization_id, cost_preset_id) VALUES (?,?,?,?)`, item, budgetID, orgA, itemPreset).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, a, orgA, 100.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, b, orgA, 100.0).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return repo.DeductForCompletedBudget(context.Background(), tx, budgetID, orgA, "sys")
	}))
	return a, b
}

func insertCostPreset(t *testing.T, db *gorm.DB, waste float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cost_presets (id, organization_id, waste_grams_per_color_change) VALUES (?,?,?)`, id, orgA, waste).Error)
	return id
}

// TestIntegration_Deduct_WasteUsesItemPreset proves the item's own cost preset waste wins
// over the budget-level preset: a 2-filament item with an item preset of 40g/change purges
// 40g (20g per filament).
func TestIntegration_Deduct_WasteUsesItemPreset(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)

	itemPreset := insertCostPreset(t, db, 40)   // effective
	budgetPreset := insertCostPreset(t, db, 10) // ignored because the item has its own
	a, b := deductItemWithPresets(t, db, repo, &itemPreset, &budgetPreset)

	require.Equal(t, int64(880), stockOf(t, db, a), "100g + 20g purge")
	require.Equal(t, int64(880), stockOf(t, db, b), "100g + 20g purge")
}

// TestIntegration_Deduct_WasteFallsBackToBudgetPreset proves an item with no preset of its
// own uses the budget-level preset: 30g/change => 30g purge (15g per filament).
func TestIntegration_Deduct_WasteFallsBackToBudgetPreset(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)

	budgetPreset := insertCostPreset(t, db, 30)
	a, b := deductItemWithPresets(t, db, repo, nil, &budgetPreset)

	require.Equal(t, int64(885), stockOf(t, db, a), "100g + 15g purge")
	require.Equal(t, int64(885), stockOf(t, db, b), "100g + 15g purge")
}

// TestIntegration_Deduct_WasteDefaultsWhenNoPreset proves that with neither an item nor a
// budget preset the default 15g/change applies (15g purge => 7.5g per filament, rounded
// once per filament to 8g on 100g).
func TestIntegration_Deduct_WasteDefaultsWhenNoPreset(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)

	a, b := deductItemWithPresets(t, db, repo, nil, nil)

	require.Equal(t, int64(892), stockOf(t, db, a), "round(100 + 7.5) = 108")
	require.Equal(t, int64(892), stockOf(t, db, b), "round(100 + 7.5) = 108")
}

// TestIntegration_LowStockForBudget lists only the budget's tracked filaments at or
// below their threshold, and the manual movement summary carries name/color.
func TestIntegration_LowStockForBudget(t *testing.T) {
	db := openStockTestDB(t)
	repo := repoimpl.NewStockRepository(db)
	threshold := 200

	low := insertFilament(t, db, orgA, 150, true, &threshold)
	ok := insertFilament(t, db, orgA, 900, true, &threshold)
	untracked := insertFilament(t, db, orgA, 10, false, &threshold)
	noThreshold := insertFilament(t, db, orgA, 10, true, nil)
	_ = insertFilament(t, db, orgA, 10, true, &threshold) // not in the budget

	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, quote_number) VALUES (?,?,?)`, budgetID, orgA, 9).Error)
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, organization_id) VALUES (?,?,?)`, item, budgetID, orgA).Error)
	for _, f := range []uuid.UUID{low, low, ok, untracked, noThreshold} {
		require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, f, orgA, 10.0).Error)
	}

	got, err := repo.LowStockForBudget(context.Background(), budgetID, orgA)
	require.NoError(t, err)
	require.Len(t, got, 1, "deduplicated and filtered")
	require.Equal(t, low, got[0].ID)
	require.Equal(t, "PLA", got[0].Name)
	require.Equal(t, "red", got[0].Color)
	require.Equal(t, int64(150), got[0].StockGrams)

	_, summary, err := repo.CreateManualMovement(context.Background(), &entities.StockMovementEntity{
		OrganizationID: orgA, FilamentID: ok, Type: entities.MovementWaste, Grams: -750, CreatedBy: "u1",
	})
	require.NoError(t, err)
	require.Equal(t, "PLA", summary.Name)
	require.True(t, summary.IsLowStock)
}
