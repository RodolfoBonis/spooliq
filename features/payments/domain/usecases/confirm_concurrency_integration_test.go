package usecases_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	paymentModels "github.com/RodolfoBonis/spooliq/features/payments/data/models"
	repoimpl "github.com/RodolfoBonis/spooliq/features/payments/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// concurrencyDB opens a multi-connection DB pinned to a fresh schema (via DSN
// search_path) and migrates the payments + minimal budgets/customers tables. It
// returns the db and a seeded budget (total 10000) with two pending 5000 payments.
func concurrencyDB(t *testing.T, org string) (*gorm.DB, uuid.UUID) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping payments concurrency test")
	}
	boot, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	schema := fmt.Sprintf("payconc_%s", uuid.New().String()[:8])
	require.NoError(t, boot.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	t.Cleanup(func() {
		_ = boot.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		if sb, e := boot.DB(); e == nil {
			_ = sb.Close()
		}
	})

	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+sep+"options=-c%20search_path%3D"+schema), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, db.AutoMigrate(&paymentModels.PaymentIntegrationModel{}, &paymentModels.BudgetPaymentModel{}))
	require.NoError(t, db.Exec(`CREATE TABLE budgets (
		id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, customer_id uuid,
		public_token varchar(43), status varchar(20) NOT NULL DEFAULT 'approved',
		total_cost bigint NOT NULL DEFAULT 0, amount_paid_cents bigint NOT NULL DEFAULT 0,
		amount_refund_due_cents bigint NOT NULL DEFAULT 0, deposit_percent double precision,
		payment_status varchar(10) NOT NULL DEFAULT 'unpaid', paid_at timestamptz,
		valid_until timestamptz, name varchar(255) NOT NULL DEFAULT '', quote_number integer,
		updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE customers (id uuid PRIMARY KEY, organization_id varchar(255) NOT NULL, name varchar(255) NOT NULL DEFAULT '', email varchar(255), phone varchar(50))`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_bp_order ON budget_payments(order_nsu)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_bp_pending ON budget_payments(budget_id, kind) WHERE status = 'pending'`).Error)

	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()
	box := testBox()
	key, _ := box.EncryptString("sk_live_testkey1234")
	tok, _ := box.EncryptString("webhooktoken")
	require.NoError(t, repo.CreateIntegration(ctx, &entities.PaymentIntegration{
		ID: uuid.New(), OrganizationID: org, Provider: entities.ProviderInfinitePay,
		Handle: "store", APIKeyCiphertext: key, WebhookTokenCiphertext: tok, Enabled: true, AssuredInstallments: 1,
	}))
	budgetID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO budgets (id, organization_id, total_cost, name) VALUES (?,?,?,?)`, budgetID, org, 10000, "Vaso").Error)
	for _, kv := range []struct {
		kind  entities.ChargeKind
		order string
	}{{entities.KindDeposit, "spq_a"}, {entities.KindBalance, "spq_b"}} {
		require.NoError(t, repo.CreatePayment(ctx, &entities.BudgetPayment{
			OrganizationID: org, BudgetID: budgetID, Kind: kv.kind, OrderNSU: kv.order,
			Handle: "store", AmountCents: 5000, Status: entities.PaymentPending,
		}))
	}
	return db, budgetID
}

// TestIntegration_ConcurrentConfirmsDoNotLoseAggregate is the end-to-end smoke test:
// two concurrent 5000 confirms on a 10000 budget settle to 10000/paid.
func TestIntegration_ConcurrentConfirmsDoNotLoseAggregate(t *testing.T) {
	org := "org-conc"
	db, budgetID := concurrencyDB(t, org)
	repo := repoimpl.NewPaymentRepository(db)
	box := testBox()
	client := &fakeClient{checkResp: &coreServices.InfinitePayPaymentCheckResponse{Success: true, Paid: true, PaidAmount: 5000}}
	svc := usecases.NewConfirmService(usecases.NewService(repo, client, box, testConfig(), nil, nil, nil, logger.NewLogger("test")))

	var wg sync.WaitGroup
	for _, order := range []string{"spq_a", "spq_b"} {
		wg.Add(1)
		go func(o string) {
			defer wg.Done()
			_, _ = svc.Confirm(context.Background(), org, entities.WebhookPayload{OrderNSU: o}, "", entities.SourceWebhook)
		}(order)
	}
	wg.Wait()

	var paid int64
	var status string
	require.NoError(t, db.Raw(`SELECT amount_paid_cents, payment_status FROM budgets WHERE id = ?`, budgetID).Row().Scan(&paid, &status))
	require.Equal(t, int64(10000), paid)
	require.Equal(t, "paid", status)
}

// TestIntegration_BudgetLockSerializesRecalc is the DETERMINISTIC proof: with tx A
// holding the budget lock and B blocked on LockBudget, committing A lets B's recalc
// (a fresh snapshot) see A's committed payment, so the aggregate reaches 10000.
func TestIntegration_BudgetLockSerializesRecalc(t *testing.T) {
	org := "org-lock"
	db, budgetID := concurrencyDB(t, org)
	ctx := context.Background()

	// Resolve the two payment ids.
	var idAStr, idBStr string
	require.NoError(t, db.Raw(`SELECT id::text FROM budget_payments WHERE order_nsu = 'spq_a'`).Scan(&idAStr).Error)
	require.NoError(t, db.Raw(`SELECT id::text FROM budget_payments WHERE order_nsu = 'spq_b'`).Scan(&idBStr).Error)
	idA := uuid.MustParse(idAStr)
	idB := uuid.MustParse(idBStr)

	paidFields := func() map[string]interface{} {
		return map[string]interface{}{"status": "paid", "paid_amount_cents": int64(5000), "paid_at": time.Now(), "status_reason": ""}
	}

	txA := db.Begin()
	require.NoError(t, txA.Error)
	repoA := repoimpl.NewPaymentRepository(txA)
	require.NoError(t, repoA.LockBudget(ctx, budgetID, org)) // A holds the budget lock
	_, err := repoA.MarkPaidGuarded(ctx, idA, org, paidFields())
	require.NoError(t, err)
	require.NoError(t, repoA.RecalcBudgetAggregate(ctx, budgetID, org))

	errCh := make(chan error, 1)
	go func() {
		txB := db.Begin()
		if txB.Error != nil {
			errCh <- txB.Error
			return
		}
		repoB := repoimpl.NewPaymentRepository(txB)
		if e := repoB.LockBudget(ctx, budgetID, org); e != nil { // blocks until A commits
			txB.Rollback()
			errCh <- e
			return
		}
		if _, e := repoB.MarkPaidGuarded(ctx, idB, org, paidFields()); e != nil {
			txB.Rollback()
			errCh <- e
			return
		}
		if e := repoB.RecalcBudgetAggregate(ctx, budgetID, org); e != nil {
			txB.Rollback()
			errCh <- e
			return
		}
		errCh <- txB.Commit().Error
	}()

	time.Sleep(500 * time.Millisecond) // ensure B is blocked on LockBudget
	require.NoError(t, txA.Commit().Error)
	require.NoError(t, <-errCh)

	var paid int64
	require.NoError(t, db.Raw(`SELECT amount_paid_cents FROM budgets WHERE id = ?`, budgetID).Scan(&paid).Error)
	require.Equal(t, int64(10000), paid, "B must see A's committed payment after the lock serializes them")
}
