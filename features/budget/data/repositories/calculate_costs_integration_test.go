package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to TEST_DATABASE_URL, pins a single connection and creates an
// isolated schema for the test, dropping it (and closing the connection) on cleanup.
// It returns the gorm.DB and a helper that reports whether TEST_DATABASE_URL is set.
func openTestDB(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping budget pricing integration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err, "connect to TEST_DATABASE_URL")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1) // pin so SET search_path persists for the whole test

	schema := fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})

	return db
}

// createPricingSchema creates the minimal set of tables the pricing path reads and
// writes. Columns match the GORM models (so SELECT * works) plus the rate tables
// read by the batched loaders. No foreign keys are needed; the test is self-contained.
func createPricingSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Exec(`
		CREATE TABLE budgets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			name varchar(255) NOT NULL,
			description text,
			customer_id uuid NOT NULL,
			status varchar(20) NOT NULL,
			print_time_hours integer NOT NULL DEFAULT 0,
			print_time_minutes integer NOT NULL DEFAULT 0,
			machine_preset_id uuid,
			energy_preset_id uuid,
			cost_preset_id uuid,
			include_energy_cost boolean DEFAULT false,
			include_waste_cost boolean DEFAULT false,
			filament_cost bigint DEFAULT 0,
			waste_cost bigint DEFAULT 0,
			energy_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0,
			labor_cost bigint DEFAULT 0,
			overhead_cost bigint DEFAULT 0,
			profit_amount bigint DEFAULT 0,
			total_cost bigint DEFAULT 0,
			delivery_days integer,
			payment_terms text,
			notes text,
			pdf_url varchar(500),
			owner_user_id varchar(255) NOT NULL,
			created_at timestamptz,
			updated_at timestamptz,
			deleted_at timestamptz
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE budget_items (
			id uuid PRIMARY KEY,
			budget_id uuid NOT NULL,
			filament_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL DEFAULT 0,
			"order" integer NOT NULL DEFAULT 0,
			product_name varchar(255) NOT NULL,
			product_description text,
			product_quantity integer NOT NULL DEFAULT 0,
			unit_price bigint NOT NULL DEFAULT 0,
			product_dimensions varchar(100),
			print_time_hours integer DEFAULT 0,
			print_time_minutes integer DEFAULT 0,
			setup_time_minutes integer DEFAULT 0,
			manual_labor_minutes_total integer DEFAULT 0,
			cost_preset_id uuid,
			additional_notes text,
			filament_cost bigint DEFAULT 0,
			waste_cost bigint DEFAULT 0,
			energy_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0,
			manual_labor_cost bigint DEFAULT 0,
			item_total_cost bigint DEFAULT 0,
			created_at timestamptz,
			updated_at timestamptz
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE budget_item_filaments (
			id uuid PRIMARY KEY,
			budget_item_id uuid NOT NULL,
			filament_id uuid NOT NULL,
			organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL DEFAULT 0,
			"order" integer NOT NULL DEFAULT 1,
			created_at timestamptz,
			updated_at timestamptz
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE filaments (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			price_per_kg double precision NOT NULL DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE cost_presets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			labor_cost_per_hour double precision DEFAULT 0,
			overhead_percentage double precision DEFAULT 0,
			profit_margin_percentage double precision DEFAULT 0
		)
	`).Error)
}

// TestIntegration_CalculateCosts_MultiItem exercises the REAL repository against a
// real PostgreSQL: it builds a two-item, multi-color budget and asserts the
// persisted per-item and budget costs exactly match the pure pricing engine's
// output for the same inputs (proving the batched loads + persistence wiring agree
// with services.Calculate).
func TestIntegration_CalculateCosts_MultiItem(t *testing.T) {
	db := openTestDB(t, "budget_calc")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	// Rates.
	fil1 := uuid.New()
	fil2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil1, org, 10000.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil2, org, 20000.0).Error)

	costPreset := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cost_presets (id, organization_id, labor_cost_per_hour, overhead_percentage, profit_margin_percentage) VALUES (?,?,?,?,?)`,
		costPreset, org, 60.0, 10.0, 20.0).Error)

	// Budget: waste on, energy off, no budget-level cost preset.
	budgetID := uuid.New()
	customerID := uuid.New()
	require.NoError(t, repo.Create(ctx, &entities.BudgetEntity{
		ID:               budgetID,
		OrganizationID:   org,
		Name:             "Multi-item",
		CustomerID:       customerID,
		Status:           entities.StatusDraft,
		IncludeWasteCost: true,
		OwnerUserID:      "user-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}))

	// Item A: multi-color (2 filaments), setup + labor, item-level cost preset.
	itemA := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID:                      itemA,
		BudgetID:                budgetID,
		FilamentID:              fil1,
		OrganizationID:          org,
		Quantity:                150,
		Order:                   1,
		ProductName:             "A",
		ProductQuantity:         1,
		SetupTimeMinutes:        30,
		ManualLaborMinutesTotal: 60,
		CostPresetID:            &costPreset,
		CreatedAt:               now,
		UpdatedAt:               now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemA, FilamentID: fil1, OrganizationID: org, Quantity: 100, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemA, FilamentID: fil2, OrganizationID: org, Quantity: 50, Order: 2, CreatedAt: now, UpdatedAt: now,
	}))

	// Item B: single color, qty 2, no cost preset.
	itemB := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID:              itemB,
		BudgetID:        budgetID,
		FilamentID:      fil1,
		OrganizationID:  org,
		Quantity:        200,
		Order:           2,
		ProductName:     "B",
		ProductQuantity: 2,
		CreatedAt:       now,
		UpdatedAt:       now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemB, FilamentID: fil1, OrganizationID: org, Quantity: 200, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))

	// Run the real calculation.
	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))

	// Expected: the pure engine over the same inputs.
	cp := &pricing.CostPresetInput{LaborRatePerHour: 60, OverheadPercentage: 10, ProfitMarginPercentage: 20}
	want, err := pricing.Calculate(pricing.PricingInput{
		IncludeWasteCost: true,
		Items: []pricing.PricingItemInput{
			{
				Quantity: 1, SetupTimeMinutes: 30, ManualLaborMinutesTotal: 60, CostPreset: cp,
				Filaments: []pricing.PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}, {Grams: 50, PricePerKgCents: 20000}},
			},
			{
				Quantity:  2,
				Filaments: []pricing.PricingFilamentInput{{Grams: 200, PricePerKgCents: 10000}},
			},
		},
	})
	require.NoError(t, err)

	// Assert budget totals persisted.
	var gotBudget struct {
		FilamentCost, WasteCost, EnergyCost, SetupCost, LaborCost, OverheadCost, ProfitAmount, TotalCost int64
	}
	require.NoError(t, db.Raw(`SELECT filament_cost, waste_cost, energy_cost, setup_cost, labor_cost, overhead_cost, profit_amount, total_cost FROM budgets WHERE id = ?`, budgetID).Scan(&gotBudget).Error)
	require.Equal(t, want.FilamentCost, gotBudget.FilamentCost, "budget filament_cost")
	require.Equal(t, want.WasteCost, gotBudget.WasteCost, "budget waste_cost")
	require.Equal(t, want.EnergyCost, gotBudget.EnergyCost, "budget energy_cost")
	require.Equal(t, want.SetupCost, gotBudget.SetupCost, "budget setup_cost")
	require.Equal(t, want.LaborCost, gotBudget.LaborCost, "budget labor_cost")
	require.Equal(t, want.Overhead, gotBudget.OverheadCost, "budget overhead_cost")
	require.Equal(t, want.Profit, gotBudget.ProfitAmount, "budget profit_amount")
	require.Equal(t, want.Total, gotBudget.TotalCost, "budget total_cost")

	// Assert per-item costs persisted (ordered by "order").
	type itemRow struct {
		FilamentCost, WasteCost, EnergyCost, SetupCost, ManualLaborCost, ItemTotalCost, UnitPrice int64
	}
	var rows []itemRow
	require.NoError(t, db.Raw(`SELECT filament_cost, waste_cost, energy_cost, setup_cost, manual_labor_cost, item_total_cost, unit_price FROM budget_items WHERE budget_id = ? ORDER BY "order" ASC`, budgetID).Scan(&rows).Error)
	require.Len(t, rows, 2)
	for i, r := range rows {
		require.Equal(t, want.Items[i].FilamentCost, r.FilamentCost, "item %d filament_cost", i)
		require.Equal(t, want.Items[i].WasteCost, r.WasteCost, "item %d waste_cost", i)
		require.Equal(t, want.Items[i].EnergyCost, r.EnergyCost, "item %d energy_cost", i)
		require.Equal(t, want.Items[i].SetupCost, r.SetupCost, "item %d setup_cost", i)
		require.Equal(t, want.Items[i].ManualLaborCost, r.ManualLaborCost, "item %d manual_labor_cost", i)
		require.Equal(t, want.Items[i].ItemTotalCost, r.ItemTotalCost, "item %d item_total_cost", i)
		require.Equal(t, want.Items[i].UnitCost, r.UnitPrice, "item %d unit_price (cost)", i)
	}
}

// TestIntegration_ComputeBudgetPricing_DoesNotPersist proves the stateless pricing
// computation used by the preview endpoint reads the org-scoped rates and returns a
// correct result WITHOUT writing any budget or item rows.
func TestIntegration_ComputeBudgetPricing_DoesNotPersist(t *testing.T) {
	db := openTestDB(t, "budget_preview")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 12000.0).Error)

	in := entities.PricingComputationInput{
		OrganizationID: org,
		Items: []entities.PricingItemSpec{
			{
				ProductQuantity: 1,
				Filaments:       []entities.PricingFilamentSpec{{FilamentID: fil, Quantity: 100}},
			},
		},
	}

	result, err := repo.ComputeBudgetPricing(ctx, in)
	require.NoError(t, err)
	require.Equal(t, int64(1200), result.FilamentCost, "100g @ R$120/kg => R$12,00")
	require.Equal(t, int64(1200), result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(1200), result.Items[0].SaleTotal)

	// The core guarantee: nothing was persisted.
	var budgetCount, itemCount, filamentLinkCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM budgets`).Scan(&budgetCount).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM budget_items`).Scan(&itemCount).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM budget_item_filaments`).Scan(&filamentLinkCount).Error)
	require.Equal(t, int64(0), budgetCount, "preview must not create budgets")
	require.Equal(t, int64(0), itemCount, "preview must not create budget items")
	require.Equal(t, int64(0), filamentLinkCount, "preview must not create item filaments")
}
