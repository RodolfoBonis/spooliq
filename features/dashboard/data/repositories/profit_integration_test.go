package repositories_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/dashboard/data/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const profitOrg = "org-p"

// openProfitDB creates the tables the profit queries read, in a throwaway schema.
func openProfitDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping dashboard profit integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("dash_profit_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})
	for _, s := range []string{
		`CREATE TABLE customers (id uuid PRIMARY KEY, organization_id varchar(255), name varchar(255), email varchar(255), created_at timestamptz DEFAULT now(), deleted_at timestamptz)`,
		`CREATE TABLE materials (id uuid PRIMARY KEY, name varchar(255))`,
		`CREATE TABLE filaments (id uuid PRIMARY KEY, name varchar(255), color varchar(100) DEFAULT '', color_hex varchar(7), material_id uuid)`,
		`CREATE TABLE presets (id uuid PRIMARY KEY, name varchar(255))`,
		`CREATE TABLE budgets (
			id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, customer_id uuid,
			name varchar(255) DEFAULT '', quote_number integer, status varchar(20) NOT NULL,
			machine_preset_id uuid, cost_preset_id uuid,
			total_cost bigint DEFAULT 0, tax_amount bigint DEFAULT 0, shipping_cost bigint DEFAULT 0,
			profit_amount bigint DEFAULT 0, discount_amount bigint DEFAULT 0,
			filament_cost bigint DEFAULT 0, waste_cost bigint DEFAULT 0, energy_cost bigint DEFAULT 0,
			machine_cost bigint DEFAULT 0, setup_cost bigint DEFAULT 0, labor_cost bigint DEFAULT 0,
			post_processing_cost bigint DEFAULT 0, packaging_cost bigint DEFAULT 0,
			quality_control_cost bigint DEFAULT 0, failure_cost bigint DEFAULT 0, overhead_cost bigint DEFAULT 0,
			valid_until timestamptz, customer_response_at timestamptz, customer_response_name varchar(120),
			rejection_reason text, approved_at timestamptz, completed_at timestamptz,
			created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now(), deleted_at timestamptz)`,
		`CREATE TABLE budget_items (id uuid PRIMARY KEY, budget_id uuid, item_total_cost bigint DEFAULT 0,
			print_time_hours integer DEFAULT 0, print_time_minutes integer DEFAULT 0)`,
		`CREATE TABLE budget_item_filaments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), budget_item_id uuid, filament_id uuid, quantity numeric)`,
		`CREATE TABLE budget_status_history (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), budget_id uuid,
			organization_id varchar(255), previous_status varchar(20), new_status varchar(20),
			changed_by varchar(255), notes text, created_at timestamptz)`,
	} {
		require.NoError(t, db.Exec(s).Error, s)
	}
	return db
}

type sale struct {
	id, customer, machine  uuid.UUID
	status                 string
	total, tax, ship       int64
	profit, discount       int64
	approved               time.Time
	printMinutes, itemCost int64
	filament               uuid.UUID
	grams                  float64
}

func insertSale(t *testing.T, db *gorm.DB, s sale) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, customer_id, status, machine_preset_id,
		total_cost, tax_amount, shipping_cost, profit_amount, discount_amount, filament_cost, overhead_cost, approved_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.id, profitOrg, s.customer, s.status, s.machine, s.total, s.tax, s.ship, s.profit, s.discount,
		s.itemCost, 0, s.approved, s.approved.Add(-48*time.Hour)).Error)
	item := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budget_items (id, budget_id, item_total_cost, print_time_hours, print_time_minutes) VALUES (?, ?, ?, ?, ?)`,
		item, s.id, s.itemCost, s.printMinutes/60, s.printMinutes%60).Error)
	require.NoError(t, db.Exec(`INSERT INTO budget_item_filaments (budget_item_id, filament_id, quantity) VALUES (?, ?, ?)`,
		item, s.filament, s.grams).Error)
}

func history(t *testing.T, db *gorm.DB, budget uuid.UUID, status string, at time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO budget_status_history (budget_id, organization_id, previous_status, new_status, changed_by, created_at)
		VALUES (?, ?, 'x', ?, 'u', ?)`, budget, profitOrg, status, at).Error)
}

func TestProfitMetrics(t *testing.T) {
	db := openProfitDB(t)
	repo := repoimpl.NewDashboardRepository(db)

	now := time.Now()
	start, end := now.Add(-30*24*time.Hour), now.Add(time.Hour)
	inWindow := now.Add(-5 * 24 * time.Hour)

	ana, beto := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO customers (id, organization_id, name, email) VALUES (?, ?, 'Ana', 'a@x'), (?, ?, 'Beto', 'b@x')`,
		ana, profitOrg, beto, profitOrg).Error)
	pla, petg := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO materials (id, name) VALUES (?, 'PLA'), (?, 'PETG')`, pla, petg).Error)
	plaBlack, petgBlue := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, name, color, material_id) VALUES (?, 'PLA Basic', 'Preto', ?), (?, 'PETG HF', 'Azul', ?)`,
		plaBlack, pla, petgBlue, petg).Error)
	a1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO presets (id, name) VALUES (?, 'Bambu A1')`, a1).Error)

	completed, approved, old, rejected := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	// Completed: R$ 120 total incl. R$ 10 tax + R$ 10 shipping, R$ 40 markup - R$ 10 discount.
	insertSale(t, db, sale{id: completed, customer: ana, machine: a1, status: "completed",
		total: 12000, tax: 1000, ship: 1000, profit: 4000, discount: 1000,
		approved: inWindow, printMinutes: 120, itemCost: 6000, filament: plaBlack, grams: 300})
	// Approved (forecast): R$ 50, R$ 10 profit, PETG.
	insertSale(t, db, sale{id: approved, customer: beto, machine: a1, status: "approved",
		total: 5000, profit: 1000, approved: inWindow, printMinutes: 60, itemCost: 4000, filament: petgBlue, grams: 100})
	// Approved long ago: outside the window (approval date drives the period).
	insertSale(t, db, sale{id: old, customer: ana, machine: a1, status: "completed",
		total: 99999, profit: 50000, approved: now.Add(-90 * 24 * time.Hour), itemCost: 1, filament: plaBlack, grams: 1})
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, customer_id, status, total_cost, created_at, rejection_reason, customer_response_at, customer_response_name)
		VALUES (?, ?, ?, 'rejected', 7000, ?, 'muito caro', ?, 'Beto')`, rejected, profitOrg, beto, inWindow, inWindow).Error)

	// Decisions: two approvals (one later completed) + one rejection in the window.
	history(t, db, completed, "sent", inWindow.Add(-24*time.Hour))
	history(t, db, completed, "approved", inWindow)
	history(t, db, completed, "completed", inWindow.Add(24*time.Hour))
	history(t, db, approved, "sent", inWindow.Add(-4*time.Hour))
	history(t, db, approved, "approved", inWindow)
	history(t, db, rejected, "sent", inWindow.Add(-48*time.Hour))
	history(t, db, rejected, "rejected", inWindow)

	t.Run("overview", func(t *testing.T) {
		o, err := repo.GetOverview(profitOrg, start, end, start.Add(-30*24*time.Hour), start)
		require.NoError(t, err)
		require.EqualValues(t, 17000, o.TotalRevenue)            // gross
		require.EqualValues(t, 15000, o.NetRevenue)              // 10000 + 5000
		require.EqualValues(t, 4000, o.Profit)                   // (4000-1000) + 1000
		require.EqualValues(t, 3000, o.ProfitRealized)           // completed only
		require.EqualValues(t, 1000, o.ProfitForecast)           // approved
		require.EqualValues(t, 11000, o.ProductionCost)          // net - profit
		require.InDelta(t, 26.67, o.ProfitMargin, 0.01)          // 4000 / 15000, weighted
		require.InDelta(t, 3.0, o.PrintHours, 0.001)             // 2h + 1h
		require.EqualValues(t, 133333/100, o.ProfitPerPrintHour) // 4000 / 3h
		require.InDelta(t, 66.67, o.ApprovalRate, 0.01)          // 2 approved / 3 decided
		require.Equal(t, 2, o.SalesCount)
	})

	t.Run("profitability", func(t *testing.T) {
		p, err := repo.GetProfitability(profitOrg, start, end, 10)
		require.NoError(t, err)
		require.Len(t, p.ByMaterial, 2)
		require.Equal(t, "PLA", p.ByMaterial[0].Name)
		require.EqualValues(t, 3000, p.ByMaterial[0].Profit)
		require.EqualValues(t, 10000, p.ByMaterial[0].Revenue)
		require.InDelta(t, 300, p.ByMaterial[0].Grams, 0.01)

		require.Equal(t, "Ana", p.ByCustomer[0].Name)
		require.InDelta(t, 1000.0/11000*100, p.ByCustomer[0].DiscountRate, 0.01)
		require.True(t, p.ByCustomer[0].Repeat, "Ana has two sales ever")
		require.False(t, p.ByCustomer[1].Repeat)

		require.Len(t, p.ByMachine, 1)
		require.Equal(t, "Bambu A1", p.ByMachine[0].Name)
		require.InDelta(t, 3.0, p.ByMachine[0].Hours, 0.001)
		require.InDelta(t, 26.67, p.AverageMargin, 0.01)
	})

	t.Run("response times", func(t *testing.T) {
		r, err := repo.GetResponseTimes(profitOrg, start, end)
		require.NoError(t, err)
		require.Equal(t, 2, r.Approved)
		require.Equal(t, 1, r.Rejected)
		require.InDelta(t, 14.0, r.ApprovalMedianHours, 0.01) // median of 24h and 4h
		require.InDelta(t, 48.0, r.RejectionMedianHours, 0.01)
		require.Equal(t, 1, r.ApprovalBuckets[0].Count) // 4h
		require.Equal(t, 1, r.ApprovalBuckets[1].Count) // 24h
		require.Len(t, r.RecentRejections, 1)
		require.Equal(t, "muito caro", r.RecentRejections[0].Reason)
	})
}
