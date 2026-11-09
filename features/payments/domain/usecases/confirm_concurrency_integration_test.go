package usecases_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

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

// TestIntegration_ConcurrentConfirmsDoNotLoseAggregate proves N1: two concurrent
// 5000 confirms on a 10000 budget settle to exactly 10000 / paid. The budget FOR
// UPDATE lock in the confirm transaction serializes the aggregate recompute.
func TestIntegration_ConcurrentConfirmsDoNotLoseAggregate(t *testing.T) {
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

	// Pin the schema via DSN options so EVERY pooled connection sees it (we need
	// several for real concurrency).
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	connDSN := dsn + sep + "options=-c%20search_path%3D" + schema
	db, err := gorm.Open(postgres.Open(connDSN), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
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
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_bp_txn ON budget_payments(transaction_nsu) WHERE transaction_nsu IS NOT NULL`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_bp_pending ON budget_payments(budget_id, kind) WHERE status = 'pending'`).Error)

	repo := repoimpl.NewPaymentRepository(db)
	ctx := context.Background()
	org := "org-conc"
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
	require.Equal(t, int64(10000), paid, "both concurrent confirms must be counted")
	require.Equal(t, "paid", status)
}
