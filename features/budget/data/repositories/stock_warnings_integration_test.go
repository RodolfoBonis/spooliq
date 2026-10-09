package repositories_test

import (
	"context"
	"testing"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createStockWarningSchema creates the minimal tables GetStockWarnings and
// GetStockWarningsForRequest read. GetStockWarnings now joins budgets + cost_presets to
// resolve each item's effective per-color-change purge waste, so those tables (and the
// cost_preset_id columns) are part of the fixture. Gated by TEST_DATABASE_URL (via
// openTestDB).
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
		`CREATE TABLE cost_presets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			waste_grams_per_color_change numeric NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
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
// filaments whose rounded requirement exceeds on-hand stock, summed across items AND
// including the apportioned color-change purge waste of the multi-filament item.
func TestIntegration_GetStockWarnings(t *testing.T) {
	db := openTestDB(t, "warn")
	createStockWarningSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	const org = "warn-org-a"
	short := warnInsertFilament(t, db, org, "Short", 100, true)        // quantity 800 -> warn
	plenty := warnInsertFilament(t, db, org, "Plenty", 5000, true)     // quantity 500 -> ok
	untracked := warnInsertFilament(t, db, org, "Untracked", 0, false) // skipped

	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, cost_preset_id) VALUES (?,?,NULL)`, budgetID, org).Error)
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, organization_id, cost_preset_id) VALUES (?,?,?,NULL)`, item, budgetID, org).Error)
	// One item with FOUR filament rows => N=4 => 3 color changes => default 15g*3 = 45g
	// purge, split equally = 11.25g per row. "short" occupies two rows (+22.5g).
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, short, org, 500.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, short, org, 300.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, plenty, org, 500.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (id, budget_item_id, filament_id, organization_id, quantity) VALUES (?,?,?,?,?)`, uuid.New(), item, untracked, org, 400.0).Error)

	warnings, err := repo.GetStockWarnings(ctx, budgetID, org)
	require.NoError(t, err)
	require.Len(t, warnings, 1, "only the tracked, under-stocked filament warns")
	require.Equal(t, short.String(), warnings[0].FilamentID)
	// 800 quantity + 22.5 purge = 822.5 -> round 823.
	require.Equal(t, int64(823), warnings[0].RequiredGrams)
	require.Equal(t, int64(100), warnings[0].AvailableGrams)

	// Another org sees nothing for this budget.
	empty, err := repo.GetStockWarnings(ctx, budgetID, "warn-org-b")
	require.NoError(t, err)
	require.Empty(t, empty)
}

// TestIntegration_GetStockWarningsForRequest proves the preview path flags tracked,
// under-stocked filaments from the request items, including the apportioned purge waste
// of the item's effective cost preset, loading the waste in one query.
func TestIntegration_GetStockWarningsForRequest(t *testing.T) {
	db := openTestDB(t, "warnreq")
	createStockWarningSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	const org = "warnreq-org"
	short := warnInsertFilament(t, db, org, "Short", 100, true)
	plenty := warnInsertFilament(t, db, org, "Plenty", 5000, true)

	// An item-level cost preset with 20g per color change.
	presetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cost_presets (id, organization_id, waste_grams_per_color_change) VALUES (?,?,?)`, presetID, org, 20.0).Error)

	// One item, two filaments, the 20g preset => 1 change => 20g => 10g per row.
	items := []entities.PricingItemSpec{{
		CostPresetID: &presetID,
		Filaments: []entities.PricingFilamentSpec{
			{FilamentID: short, Quantity: 450.4},
			{FilamentID: plenty, Quantity: 500},
		},
	}}

	warnings, err := repo.GetStockWarningsForRequest(ctx, org, items, nil)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Equal(t, short.String(), warnings[0].FilamentID)
	// 450.4 + 10 purge = 460.4 -> round 460.
	require.Equal(t, int64(460), warnings[0].RequiredGrams, "required grams include waste, rounded")
	require.Equal(t, int64(100), warnings[0].AvailableGrams)

	// Empty input -> empty, non-nil.
	none, err := repo.GetStockWarningsForRequest(ctx, org, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, none)
	require.Empty(t, none)
}
