package repositories_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/data/models"
	repoimpl "github.com/RodolfoBonis/spooliq/features/payments/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDB connects to TEST_DATABASE_URL and creates an isolated schema for the
// test, dropping it on cleanup.
func openTestDB(t *testing.T, prefix string) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payments integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})
	return db
}

// createPaymentsSchema migrates the payment tables plus the minimal budgets/customers
// tables GetBudgetChargeContext reads, then creates the unique indexes that enforce
// the money-safety invariants (mirrors RunPaymentsMigrations).
func createPaymentsSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&models.PaymentIntegrationModel{},
		&models.BudgetPaymentModel{},
		&models.PaymentWebhookEventModel{},
	))
	require.NoError(t, db.Exec(`CREATE TABLE budgets (
		id uuid PRIMARY KEY,
		organization_id varchar(255) NOT NULL,
		customer_id uuid,
		public_token varchar(43),
		status varchar(20) NOT NULL DEFAULT 'approved',
		total_cost bigint NOT NULL DEFAULT 0,
		amount_paid_cents bigint NOT NULL DEFAULT 0,
		amount_refund_due_cents bigint NOT NULL DEFAULT 0,
		deposit_percent double precision,
		payment_status varchar(10) NOT NULL DEFAULT 'unpaid',
		paid_at timestamptz,
		valid_until timestamptz,
		name varchar(255) NOT NULL DEFAULT '',
		quote_number integer,
		updated_at timestamptz NOT NULL DEFAULT now(),
		deleted_at timestamptz
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE customers (
		id uuid PRIMARY KEY,
		organization_id varchar(255) NOT NULL,
		name varchar(255) NOT NULL DEFAULT '',
		email varchar(255),
		phone varchar(50)
	)`).Error)

	idx := []string{
		"CREATE UNIQUE INDEX uq_budget_payments_order_nsu ON budget_payments(order_nsu)",
		"CREATE UNIQUE INDEX uq_budget_payments_transaction_nsu ON budget_payments(transaction_nsu) WHERE transaction_nsu IS NOT NULL",
		"CREATE UNIQUE INDEX uq_budget_payments_pending ON budget_payments(budget_id, kind) WHERE status = 'pending'",
	}
	for _, s := range idx {
		require.NoError(t, db.Exec(s).Error, s)
	}
}

func newPayment(org string, budgetID uuid.UUID, kind entities.ChargeKind, orderNSU string, amount int64, status entities.PaymentRecordStatus) *entities.BudgetPayment {
	return &entities.BudgetPayment{
		ID: uuid.New(), OrganizationID: org, BudgetID: budgetID, Kind: kind,
		OrderNSU: orderNSU, AmountCents: amount, Status: status,
	}
}

func TestIntegration_PendingUniqueIndex(t *testing.T) {
	db := openTestDB(t, "pay_pending")
	createPaymentsSchema(t, db)
	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()
	budgetID := uuid.New()

	require.NoError(t, repo.CreatePayment(ctx, newPayment("org1", budgetID, entities.KindFull, "spq_1", 2000, entities.PaymentPending)))
	err := repo.CreatePayment(ctx, newPayment("org1", budgetID, entities.KindFull, "spq_2", 2000, entities.PaymentPending))
	require.Error(t, err, "second pending link for the same (budget, kind) must be rejected")
}

func TestIntegration_OrderNSUUniqueIndex(t *testing.T) {
	db := openTestDB(t, "pay_order")
	createPaymentsSchema(t, db)
	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.CreatePayment(ctx, newPayment("org1", uuid.New(), entities.KindFull, "spq_dup", 2000, entities.PaymentPending)))
	err := repo.CreatePayment(ctx, newPayment("org1", uuid.New(), entities.KindDeposit, "spq_dup", 1000, entities.PaymentPending))
	require.Error(t, err, "duplicate order_nsu must be rejected")
}

func TestIntegration_GuardedMarkPaidAndAggregate(t *testing.T) {
	db := openTestDB(t, "pay_paid")
	createPaymentsSchema(t, db)
	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()

	budgetID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO budgets (id, organization_id, total_cost) VALUES (?,?,?)", budgetID, "org1", 2000).Error)
	p := newPayment("org1", budgetID, entities.KindFull, "spq_guard", 2000, entities.PaymentPending)
	require.NoError(t, repo.CreatePayment(ctx, p))

	rows, err := repo.MarkPaidGuarded(ctx, p.ID, "org1", map[string]interface{}{
		"status": string(entities.PaymentPaid), "paid_amount_cents": int64(2000), "paid_at": time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rows, "first guarded mark should affect one row")

	rows, err = repo.MarkPaidGuarded(ctx, p.ID, "org1", map[string]interface{}{"status": string(entities.PaymentPaid)})
	require.NoError(t, err)
	require.Equal(t, int64(0), rows, "second guarded mark is a no-op (idempotency)")

	sum, err := repo.SumPaidPrincipal(ctx, budgetID)
	require.NoError(t, err)
	require.Equal(t, int64(2000), sum)

	require.NoError(t, repo.RecalcBudgetAggregate(ctx, budgetID, "org1"))

	fresh, err := repo.GetBudgetChargeContext(ctx, budgetID, "org1")
	require.NoError(t, err)
	require.Equal(t, int64(2000), fresh.AmountPaidCents)
}

func TestIntegration_CancelPendingForBudgetInTransaction(t *testing.T) {
	db := openTestDB(t, "pay_cancel")
	createPaymentsSchema(t, db)
	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()
	budgetID := uuid.New()

	require.NoError(t, repo.CreatePayment(ctx, newPayment("org1", budgetID, entities.KindDeposit, "spq_c1", 1000, entities.PaymentPending)))

	err := repo.WithTransaction(ctx, func(tx domainRepos.PaymentRepository) error {
		return tx.CancelPendingForBudget(ctx, budgetID, "org1", entities.ReasonBudgetRevised)
	})
	require.NoError(t, err)

	got, err := repo.GetPaymentByOrderNSU(ctx, "org1", "spq_c1")
	require.NoError(t, err)
	require.Equal(t, entities.PaymentCanceled, got.Status)
	require.Equal(t, entities.ReasonBudgetRevised, got.StatusReason)
}
