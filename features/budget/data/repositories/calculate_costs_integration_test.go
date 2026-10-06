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
			profile_id uuid,
			machine_preset_id uuid,
			energy_preset_id uuid,
			cost_preset_id uuid,
			include_energy_cost boolean DEFAULT false,
			include_waste_cost boolean DEFAULT false,
			include_machine_cost boolean DEFAULT true,
			discount_type varchar(10),
			discount_value double precision,
			include_shipping boolean DEFAULT false,
			shipping_override bigint,
			tax_rate double precision,
			filament_cost bigint DEFAULT 0,
			waste_cost bigint DEFAULT 0,
			energy_cost bigint DEFAULT 0,
			machine_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0,
			labor_cost bigint DEFAULT 0,
			post_processing_cost bigint DEFAULT 0,
			packaging_cost bigint DEFAULT 0,
			quality_control_cost bigint DEFAULT 0,
			failure_cost bigint DEFAULT 0,
			overhead_cost bigint DEFAULT 0,
			profit_amount bigint DEFAULT 0,
			discount_amount bigint DEFAULT 0,
			shipping_cost bigint DEFAULT 0,
			tax_amount bigint DEFAULT 0,
			tax_rate_applied double precision DEFAULT 0,
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
			post_processing_minutes integer DEFAULT 0,
			support_removal_minutes integer DEFAULT 0,
			cost_preset_id uuid,
			additional_notes text,
			model_3d_id uuid,
			filament_cost bigint DEFAULT 0,
			waste_cost bigint DEFAULT 0,
			energy_cost bigint DEFAULT 0,
			machine_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0,
			manual_labor_cost bigint DEFAULT 0,
			post_processing_cost bigint DEFAULT 0,
			support_removal_cost bigint DEFAULT 0,
			packaging_cost bigint DEFAULT 0,
			quality_control_cost bigint DEFAULT 0,
			failure_cost bigint DEFAULT 0,
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
			profit_margin_percentage double precision DEFAULT 0,
			post_processing_cost_per_hour double precision DEFAULT 0,
			support_removal_cost_per_hour double precision DEFAULT 0,
			packaging_cost_per_item double precision DEFAULT 0,
			quality_control_cost_per_item double precision DEFAULT 0,
			shipping_cost_base double precision DEFAULT 0,
			shipping_cost_per_gram double precision DEFAULT 0,
			failure_rate_percentage double precision DEFAULT 0,
			waste_grams_per_color_change double precision DEFAULT 15
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE machine_presets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			power_consumption double precision DEFAULT 0,
			cost_per_hour double precision DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE energy_presets (
			id uuid PRIMARY KEY,
			organization_id varchar(255) NOT NULL,
			energy_cost_per_kwh double precision DEFAULT 0
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE companies (
			id uuid DEFAULT gen_random_uuid(),
			organization_id varchar(255) PRIMARY KEY,
			default_tax_rate double precision NOT NULL DEFAULT 0,
			deleted_at timestamptz
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

// TestIntegration_BudgetCostPresetDrivesOverheadAndItemFallback proves two Phase-2b
// behaviours end-to-end against a real PostgreSQL:
//
//  1. the budget-level cost preset (stored on the budget) drives overhead/profit; and
//  2. an item WITHOUT its own cost_preset_id falls back to the budget-level cost
//     preset for its setup/labor rates — without that NULL being mutated on the row.
//
// The persisted costs must match the pure engine fed the same budget-level preset as
// BOTH the overhead/profit source and the (fallback) item cost preset.
func TestIntegration_BudgetCostPresetDrivesOverheadAndItemFallback(t *testing.T) {
	db := openTestDB(t, "budget_costfallback")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 10000.0).Error)

	// Budget-level cost preset: labor R$90/h, 15% overhead, 25% profit.
	budgetCostPreset := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cost_presets (id, organization_id, labor_cost_per_hour, overhead_percentage, profit_margin_percentage) VALUES (?,?,?,?,?)`,
		budgetCostPreset, org, 90.0, 15.0, 25.0).Error)

	budgetID := uuid.New()
	require.NoError(t, repo.Create(ctx, &entities.BudgetEntity{
		ID:             budgetID,
		OrganizationID: org,
		Name:           "Cost fallback",
		CustomerID:     uuid.New(),
		Status:         entities.StatusDraft,
		CostPresetID:   &budgetCostPreset, // budget-level cost preset
		OwnerUserID:    "user-1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}))

	// Single item with NO cost_preset_id but with setup + manual labor minutes.
	itemID := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID:                      itemID,
		BudgetID:                budgetID,
		FilamentID:              fil,
		OrganizationID:          org,
		Quantity:                100,
		Order:                   1,
		ProductName:             "A",
		ProductQuantity:         1,
		SetupTimeMinutes:        40,
		ManualLaborMinutesTotal: 20,
		CostPresetID:            nil, // must fall back to the budget-level preset
		CreatedAt:               now,
		UpdatedAt:               now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org, Quantity: 100, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))

	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))

	// Expected: the pure engine with the budget preset as BOTH the overhead/profit
	// source and the item's (fallback) cost preset.
	cp := &pricing.CostPresetInput{LaborRatePerHour: 90, OverheadPercentage: 15, ProfitMarginPercentage: 25}
	want, err := pricing.Calculate(pricing.PricingInput{
		BudgetCostPreset: cp,
		Items: []pricing.PricingItemInput{
			{
				Quantity: 1, SetupTimeMinutes: 40, ManualLaborMinutesTotal: 20, CostPreset: cp,
				Filaments: []pricing.PricingFilamentInput{{Grams: 100, PricePerKgCents: 10000}},
			},
		},
	})
	require.NoError(t, err)

	// The fallback must have produced non-zero setup/labor (proving it was applied).
	require.Greater(t, want.SetupCost, int64(0))
	require.Greater(t, want.LaborCost, int64(0))

	var gotBudget struct {
		SetupCost, LaborCost, OverheadCost, ProfitAmount, TotalCost int64
	}
	require.NoError(t, db.Raw(`SELECT setup_cost, labor_cost, overhead_cost, profit_amount, total_cost FROM budgets WHERE id = ?`, budgetID).Scan(&gotBudget).Error)
	require.Equal(t, want.SetupCost, gotBudget.SetupCost, "item setup uses budget-level labor rate")
	require.Equal(t, want.LaborCost, gotBudget.LaborCost, "item labor uses budget-level labor rate")
	require.Equal(t, want.Overhead, gotBudget.OverheadCost, "overhead from budget-level cost preset")
	require.Equal(t, want.Profit, gotBudget.ProfitAmount, "profit from budget-level cost preset")
	require.Equal(t, want.Total, gotBudget.TotalCost)

	// The stored item cost_preset_id must remain NULL (fallback never mutates the row).
	var nullCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM budget_items WHERE id = ? AND cost_preset_id IS NULL`, itemID).Scan(&nullCount).Error)
	require.Equal(t, int64(1), nullCount, "fallback must not persist a cost_preset_id on the item")
}

// TestIntegration_CalculateCosts_Phase4AColumns exercises the full Phase-4A cost
// model end-to-end against a real PostgreSQL: machine cost (include_machine_cost),
// packaging/QC/failure from the cost preset, a percent discount, computed shipping
// and the company-default "por dentro" tax. It asserts the persisted budget columns
// (including the new ones) exactly match the pure pricing engine fed the equivalent
// inputs, and that item sale totals + shipping reconcile to the stored total.
func TestIntegration_CalculateCosts_Phase4AColumns(t *testing.T) {
	db := openTestDB(t, "budget_phase4a")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 10000.0).Error)

	// Cost preset: no labor/overhead/profit, but packaging R$2/item, QC R$1/item,
	// 10% failure rate, shipping R$10 base + R$0,05/g, waste 15g default.
	costPreset := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cost_presets
		(id, organization_id, labor_cost_per_hour, overhead_percentage, profit_margin_percentage,
		 packaging_cost_per_item, quality_control_cost_per_item, failure_rate_percentage,
		 shipping_cost_base, shipping_cost_per_gram, waste_grams_per_color_change)
		VALUES (?,?,0,0,0,2,1,10,10,0.05,15)`, costPreset, org).Error)

	// Machine preset: R$10/h.
	machinePreset := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO machine_presets (id, organization_id, power_consumption, cost_per_hour) VALUES (?,?,?,?)`,
		machinePreset, org, 200.0, 10.0).Error)

	// Company default tax rate 6% (budget leaves tax_rate NULL).
	require.NoError(t, db.Exec(`INSERT INTO companies (organization_id, default_tax_rate) VALUES (?, ?)`, org, 6.0).Error)

	discountType := "percent"
	discountValue := 10.0
	budgetID := uuid.New()
	require.NoError(t, repo.Create(ctx, &entities.BudgetEntity{
		ID:                 budgetID,
		OrganizationID:     org,
		Name:               "Phase4A",
		CustomerID:         uuid.New(),
		Status:             entities.StatusDraft,
		MachinePresetID:    &machinePreset,
		CostPresetID:       &costPreset,
		IncludeMachineCost: true,
		DiscountType:       &discountType,
		DiscountValue:      &discountValue,
		IncludeShipping:    true,
		OwnerUserID:        "user-1",
		CreatedAt:          now,
		UpdatedAt:          now,
	}))

	itemID := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID: itemID, BudgetID: budgetID, FilamentID: fil, OrganizationID: org,
		Quantity: 1000, Order: 1, ProductName: "A", ProductQuantity: 1,
		PrintTimeHours: 1, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org, Quantity: 1000, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))

	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))

	// Expected: the pure engine fed the equivalent resolved inputs.
	cp := &pricing.CostPresetInput{
		PackagingCostPerItem:      2,
		QualityControlCostPerItem: 1,
		FailureRatePercentage:     10,
		WasteGramsPerColorChange:  15,
	}
	want, err := pricing.Calculate(pricing.PricingInput{
		IncludeMachineCost:  true,
		MachineCostPerHour:  10,
		BudgetCostPreset:    cp,
		DiscountType:        pricing.DiscountTypePercent,
		DiscountValue:       10,
		IncludeShipping:     true,
		ShippingCostBase:    10,
		ShippingCostPerGram: 0.05,
		TaxRatePercent:      6,
		Items: []pricing.PricingItemInput{
			{Quantity: 1, PrintTimeHours: 1, CostPreset: cp, Filaments: []pricing.PricingFilamentInput{{Grams: 1000, PricePerKgCents: 10000}}},
		},
	})
	require.NoError(t, err)
	require.Greater(t, want.MachineCost, int64(0))
	require.Greater(t, want.FailureCost, int64(0))
	require.Greater(t, want.DiscountAmount, int64(0))
	require.Greater(t, want.ShippingCost, int64(0))
	require.Greater(t, want.TaxAmount, int64(0))

	var got struct {
		MachineCost        int64
		PackagingCost      int64
		QualityControlCost int64
		FailureCost        int64
		DiscountAmount     int64
		ShippingCost       int64
		TaxAmount          int64
		TotalCost          int64
		TaxRateApplied     float64
	}
	require.NoError(t, db.Raw(`SELECT machine_cost, packaging_cost, quality_control_cost, failure_cost,
		discount_amount, shipping_cost, tax_amount, total_cost, tax_rate_applied
		FROM budgets WHERE id = ?`, budgetID).Scan(&got).Error)

	require.Equal(t, want.MachineCost, got.MachineCost, "machine_cost")
	require.Equal(t, want.PackagingCost, got.PackagingCost, "packaging_cost")
	require.Equal(t, want.QualityControlCost, got.QualityControlCost, "quality_control_cost")
	require.Equal(t, want.FailureCost, got.FailureCost, "failure_cost")
	require.Equal(t, want.DiscountAmount, got.DiscountAmount, "discount_amount")
	require.Equal(t, want.ShippingCost, got.ShippingCost, "shipping_cost")
	require.Equal(t, want.TaxAmount, got.TaxAmount, "tax_amount")
	require.Equal(t, want.Total, got.TotalCost, "total_cost")
	require.Equal(t, 6.0, got.TaxRateApplied, "tax_rate_applied = company default")

	// Item sale totals + shipping reconcile to the stored total.
	var saleSum int64
	for _, it := range want.Items {
		saleSum += it.SaleTotal
	}
	require.Equal(t, got.TotalCost-got.ShippingCost, saleSum, "item sale totals + shipping must equal total")
}
