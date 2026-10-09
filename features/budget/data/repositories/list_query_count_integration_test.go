package repositories_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// countingLogger wraps a gorm logger and counts every SQL statement traced. gorm
// calls Trace once per executed statement, regardless of log level, so this gives
// an exact per-call SQL count for the measured block.
type countingLogger struct {
	gormlogger.Interface
	count *int64
}

func (l countingLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	atomic.AddInt64(l.count, 1)
}

// createListSchema creates the tables the list response assembly reads: budgets,
// budget_items, budget_item_filaments, filaments (+ brands/materials for the JOIN),
// customers, presets and print_profiles. No foreign keys; the test is self-contained.
func createListSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL,
			description text, customer_id uuid NOT NULL, status varchar(20) NOT NULL,
			print_time_hours integer NOT NULL DEFAULT 0, print_time_minutes integer NOT NULL DEFAULT 0,
			profile_id uuid, machine_preset_id uuid, energy_preset_id uuid, cost_preset_id uuid,
			include_energy_cost boolean DEFAULT false, include_waste_cost boolean DEFAULT false,
			include_machine_cost boolean NOT NULL DEFAULT false,
			discount_type varchar(10), discount_value double precision,
			include_shipping boolean DEFAULT false, shipping_override bigint, tax_rate double precision,
			filament_cost bigint DEFAULT 0, waste_cost bigint DEFAULT 0, energy_cost bigint DEFAULT 0,
			machine_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0, labor_cost bigint DEFAULT 0,
			post_processing_cost bigint DEFAULT 0, packaging_cost bigint DEFAULT 0,
			quality_control_cost bigint DEFAULT 0, failure_cost bigint DEFAULT 0,
			overhead_cost bigint DEFAULT 0,
			profit_amount bigint DEFAULT 0,
			discount_amount bigint DEFAULT 0, shipping_cost bigint DEFAULT 0,
			tax_amount bigint DEFAULT 0, tax_rate_applied double precision DEFAULT 0,
			total_cost bigint DEFAULT 0,
			delivery_days integer, payment_terms text, notes text, pdf_url varchar(500),
quote_number integer, valid_until timestamptz, public_token varchar(43), public_token_created_at timestamptz, approved_at timestamptz, completed_at timestamptz, customer_response_at timestamptz, customer_response_name varchar(120), customer_response_ip varchar(45), customer_response_user_agent varchar(255), rejection_reason text,
						owner_user_id varchar(255) NOT NULL, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz
		)`,
		`CREATE TABLE budget_items (
			id uuid PRIMARY KEY, budget_id uuid NOT NULL, filament_id uuid NOT NULL, organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL DEFAULT 0, "order" integer NOT NULL DEFAULT 0,
			product_name varchar(255) NOT NULL, product_description text, product_quantity integer NOT NULL DEFAULT 0,
			unit_price bigint NOT NULL DEFAULT 0, product_dimensions varchar(100),
			print_time_hours integer DEFAULT 0, print_time_minutes integer DEFAULT 0,
			setup_time_minutes integer DEFAULT 0, manual_labor_minutes_total integer DEFAULT 0,
			post_processing_minutes integer DEFAULT 0, support_removal_minutes integer DEFAULT 0,
			cost_preset_id uuid, additional_notes text, model_3d_id uuid,
			filament_cost bigint DEFAULT 0, waste_cost bigint DEFAULT 0, energy_cost bigint DEFAULT 0,
			machine_cost bigint DEFAULT 0,
			setup_cost bigint DEFAULT 0, manual_labor_cost bigint DEFAULT 0,
			post_processing_cost bigint DEFAULT 0, support_removal_cost bigint DEFAULT 0,
			packaging_cost bigint DEFAULT 0, quality_control_cost bigint DEFAULT 0,
			failure_cost bigint DEFAULT 0, item_total_cost bigint DEFAULT 0,
			created_at timestamptz, updated_at timestamptz
		)`,
		`CREATE TABLE budget_item_filaments (
			id uuid PRIMARY KEY, budget_item_id uuid NOT NULL, filament_id uuid NOT NULL, organization_id varchar(255) NOT NULL,
			quantity numeric NOT NULL DEFAULT 0, "order" integer NOT NULL DEFAULT 1,
			created_at timestamptz, updated_at timestamptz
		)`,
		`CREATE TABLE filaments (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL DEFAULT '',
			color varchar(255) DEFAULT '', color_type varchar(50) DEFAULT '', color_data jsonb,
			color_hex varchar(50) DEFAULT '', color_preview varchar(255) DEFAULT '',
			price_per_kg double precision NOT NULL DEFAULT 0, brand_id uuid NOT NULL, material_id uuid NOT NULL,
			deleted_at timestamptz
		)`,
		`CREATE TABLE brands (id uuid PRIMARY KEY, name varchar(255) NOT NULL)`,
		`CREATE TABLE materials (id uuid PRIMARY KEY, name varchar(255) NOT NULL)`,
		`CREATE TABLE customers (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL,
			email varchar(255), phone varchar(50), document varchar(50)
		)`,
		`CREATE TABLE presets (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL,
			type varchar(50) NOT NULL, deleted_at timestamptz
		)`,
		`CREATE TABLE print_profiles (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL, deleted_at timestamptz
		)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
}

// seedBudgetsWithItems inserts n budgets for org, each with itemsPer items and one
// filament link per item, all sharing one customer/cost preset/profile/brand/material.
func seedBudgetsWithItems(t *testing.T, db *gorm.DB, org string, n, itemsPer int) {
	t.Helper()
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	brand := uuid.New()
	material := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO brands (id, name) VALUES (?,?)`, brand, "Brand").Error)
	require.NoError(t, db.Exec(`INSERT INTO materials (id, name) VALUES (?,?)`, material, "PLA").Error)

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, name, price_per_kg, brand_id, material_id) VALUES (?,?,?,?,?,?)`,
		fil, org, "Fil", 10000.0, brand, material).Error)

	costPreset := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO presets (id, organization_id, name, type) VALUES (?,?,?,?)`, costPreset, org, "Cost", "cost").Error)

	profile := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO print_profiles (id, organization_id, name) VALUES (?,?,?)`, profile, org, "Profile").Error)

	for b := 0; b < n; b++ {
		customer := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO customers (id, organization_id, name) VALUES (?,?,?)`, customer, org, "Customer").Error)

		budgetID := uuid.New()
		require.NoError(t, repo.Create(ctx, &entities.BudgetEntity{
			ID:             budgetID,
			OrganizationID: org,
			Name:           "Budget",
			CustomerID:     customer,
			Status:         entities.StatusDraft,
			ProfileID:      &profile,
			CostPresetID:   &costPreset,
			TotalCost:      1000,
			OwnerUserID:    "user-1",
			CreatedAt:      now.Add(time.Duration(b) * time.Second),
			UpdatedAt:      now,
		}))

		for i := 0; i < itemsPer; i++ {
			itemID := uuid.New()
			require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
				ID: itemID, BudgetID: budgetID, FilamentID: fil, OrganizationID: org,
				Quantity: 100, Order: i + 1, ProductName: "P", ProductQuantity: 1,
				CostPresetID: &costPreset, ItemTotalCost: 300, CreatedAt: now, UpdatedAt: now,
			}))
			require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
				ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org,
				Quantity: 100, Order: 1, CreatedAt: now, UpdatedAt: now,
			}))
		}
	}
}

// runListLoad executes the exact batch-loading sequence the FindAll use case runs to
// assemble a page of budgets, over a query-counting session, and returns the number
// of SQL statements executed. The sequence is: SearchBudgets (count + find), then one
// batch query each for customers, items, filament usage, cost-preset names and
// profile names.
func runListLoad(t *testing.T, db *gorm.DB, org string) int64 {
	t.Helper()
	ctx := context.Background()

	var count int64
	session := db.Session(&gorm.Session{Logger: countingLogger{Interface: gormlogger.Default.LogMode(gormlogger.Silent), count: &count}})
	repo := repoimpl.NewBudgetRepository(session)

	budgets, _, err := repo.SearchBudgets(ctx, org, map[string]interface{}{}, "created_at DESC", 100, 0)
	require.NoError(t, err)

	customerIDs := make([]uuid.UUID, 0, len(budgets))
	budgetIDs := make([]uuid.UUID, 0, len(budgets))
	profileIDs := make([]uuid.UUID, 0)
	costPresetIDs := make([]uuid.UUID, 0)
	for _, b := range budgets {
		customerIDs = append(customerIDs, b.CustomerID)
		budgetIDs = append(budgetIDs, b.ID)
		if b.ProfileID != nil {
			profileIDs = append(profileIDs, *b.ProfileID)
		}
		if b.CostPresetID != nil {
			costPresetIDs = append(costPresetIDs, *b.CostPresetID)
		}
	}

	_, err = repo.GetCustomersInfo(ctx, customerIDs, org)
	require.NoError(t, err)

	itemsByBudget, err := repo.GetItemsByBudgetIDs(ctx, budgetIDs, org)
	require.NoError(t, err)

	itemIDs := make([]uuid.UUID, 0)
	for _, items := range itemsByBudget {
		for _, item := range items {
			itemIDs = append(itemIDs, item.ID)
			if item.CostPresetID != nil {
				costPresetIDs = append(costPresetIDs, *item.CostPresetID)
			}
		}
	}

	_, err = repo.GetFilamentUsageInfoByItemIDs(ctx, itemIDs, org)
	require.NoError(t, err)
	_, err = repo.GetCostPresetNames(ctx, costPresetIDs, org)
	require.NoError(t, err)
	_, err = repo.GetProfileNames(ctx, profileIDs, org)
	require.NoError(t, err)

	return count
}

// TestIntegration_ListQueryCount_IsConstant proves the budget list page is assembled
// with a CONSTANT number of SQL statements regardless of how many budgets (and items)
// are on the page: a 5-budget × 3-item page costs exactly as many queries as a
// 2-budget × 3-item page. This is the regression guard against the former N+1
// (~4 + 2·budgets + items + presets) in the list endpoint.
func TestIntegration_ListQueryCount_IsConstant(t *testing.T) {
	const itemsPer = 3

	dbSmall := openTestDB(t, "budget_qc_small")
	createListSchema(t, dbSmall)
	seedBudgetsWithItems(t, dbSmall, "org-a", 2, itemsPer)
	small := runListLoad(t, dbSmall, "org-a")

	dbLarge := openTestDB(t, "budget_qc_large")
	createListSchema(t, dbLarge)
	seedBudgetsWithItems(t, dbLarge, "org-a", 5, itemsPer)
	large := runListLoad(t, dbLarge, "org-a")

	// The whole point: identical query count for 2 vs 5 budgets.
	require.Equal(t, small, large, "list query count must not grow with the number of budgets (N+1 regression)")

	// And it is the expected small constant: SearchBudgets (count+find) + one batch
	// query each for customers, items, filament usage, cost-preset names, profile names.
	require.Equal(t, int64(7), large, "expected a constant 7 queries per page")
}

// compile-time guard that the repository returned by NewBudgetRepository satisfies
// the interface used above.
var _ budgetRepo.BudgetRepository = repoimpl.NewBudgetRepository(nil)
