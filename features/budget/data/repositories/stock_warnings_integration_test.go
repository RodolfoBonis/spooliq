package repositories_test

import (
	"context"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createStockWarningSchema creates the minimal tables GetStockWarnings and
// GetStockWarningsForRequest read. Gated by TEST_DATABASE_URL (via openTestDB).
func createStockWarningSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL DEFAULT '',
			color varchar(100) NOT NULL DEFAULT '',
			stock_grams bigint NOT NULL DEFAULT 0,
			track_stock boolean NOT NULL DEFAULT false,
			deleted_at timestamptz
		)`,
		`CREATE TABLE budget_items (
			id uuid PRIMARY KEY,
			budget_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL
		)`,
		`CREATE TABLE budget_item_filaments (
			id uuid PRIMARY KEY,
			budget_item_id uuid NOT NULL,
			filament_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL
		)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
}

func warnInsertFilament(t *testing.T, db *gorm.DB, org, name string, stock int64, track bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, name, color, stock_grams, track_stock) VALUES (?,?,?,?,?,?)`,
		id, org, name, "red", stock, track).Error)
	return id
}

// TestIntegration_GetStockWarnings proves warnings are returned only for tracked
// filaments whose rounded requirement exceeds on-hand stock, summed across items.
func TestIntegration_GetStockWarnings(t *testing.T) {
	db := openTestDB(t, "warn")
	createStockWarningSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	const org = "warn-org-a"
	short := warnInsertFilament(t, db, org, "Short", 100, true)        // needs 800 -> warn
	plenty := warnInsertFilament(t, db, org, "Plenty", 5000, true)     // needs 500 -> ok
	untracked := warnInsertFilament(t, db, org, "Untracked", 0, false) // needs 400 -> skipped

	budgetID := uuid.New()
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, organization_id) VALUES (?,?,?)`, item, budgetID, org).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, short, org, 500.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, short, org, 300.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, plenty, org, 500.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, untracked, org, 400.0).Error)

	warnings, err := repo.GetStockWarnings(ctx, budgetID, org)
	require.NoError(t, err)
	require.Len(t, warnings, 1, "only the tracked, under-stocked filament warns")
	require.Equal(t, short.String(), warnings[0].FilamentID)
	require.Equal(t, int64(800), warnings[0].RequiredGrams)
	require.Equal(t, int64(100), warnings[0].AvailableGrams)

	// Another org sees nothing for this budget.
	empty, err := repo.GetStockWarnings(ctx, budgetID, "warn-org-b")
	require.NoError(t, err)
	require.Empty(t, empty)
}

// TestIntegration_GetStockWarningsForRequest proves the preview path flags tracked,
// under-stocked filaments from a required-grams map in one query.
func TestIntegration_GetStockWarningsForRequest(t *testing.T) {
	db := openTestDB(t, "warnreq")
	createStockWarningSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	const org = "warnreq-org"
	short := warnInsertFilament(t, db, org, "Short", 100, true)
	plenty := warnInsertFilament(t, db, org, "Plenty", 5000, true)

	req := map[uuid.UUID]float64{short: 450.4, plenty: 500}
	warnings, err := repo.GetStockWarningsForRequest(ctx, org, req)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Equal(t, short.String(), warnings[0].FilamentID)
	require.Equal(t, int64(450), warnings[0].RequiredGrams, "required grams rounded")
	require.Equal(t, int64(100), warnings[0].AvailableGrams)

	// Empty input -> empty, non-nil.
	none, err := repo.GetStockWarningsForRequest(ctx, org, map[uuid.UUID]float64{})
	require.NoError(t, err)
	require.NotNil(t, none)
	require.Empty(t, none)
}
