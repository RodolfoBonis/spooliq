package di_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	paymentModels "github.com/RodolfoBonis/spooliq/features/payments/data/models"
	paymentsDi "github.com/RodolfoBonis/spooliq/features/payments/di"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openDI(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping pending-canceler integration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("canceler_%s", uuid.New().String()[:8])
	require.NoError(t, db.Exec(fmt.Sprintf("CREATE SCHEMA %q", schema)).Error)
	require.NoError(t, db.Exec(fmt.Sprintf("SET search_path TO %q", schema)).Error)
	t.Cleanup(func() {
		_ = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schema)).Error
		_ = sqlDB.Close()
	})
	return db
}

// TestPendingCancelerCancelsInsideTransaction proves the budget status change can
// cancel pending links atomically via the tx-bound *gorm.DB.
func TestPendingCancelerCancelsInsideTransaction(t *testing.T) {
	db := openDI(t)
	require.NoError(t, db.AutoMigrate(&paymentModels.BudgetPaymentModel{}))

	budgetID := uuid.New()
	pending := paymentModels.BudgetPaymentModel{
		ID: uuid.New(), OrganizationID: "org1", BudgetID: budgetID, Kind: "deposit",
		OrderNSU: "spq_pc", AmountCents: 1000, Status: "pending",
	}
	paid := paymentModels.BudgetPaymentModel{
		ID: uuid.New(), OrganizationID: "org1", BudgetID: budgetID, Kind: "full",
		OrderNSU: "spq_paid", AmountCents: 2000, Status: "paid",
	}
	require.NoError(t, db.Create(&pending).Error)
	require.NoError(t, db.Create(&paid).Error)

	canceler := paymentsDi.NewPendingCanceler()
	err := db.Transaction(func(tx *gorm.DB) error {
		return canceler.CancelPending(context.Background(), tx, budgetID, "org1", entities.ReasonBudgetCancelled)
	})
	require.NoError(t, err)

	var gotPending, gotPaid paymentModels.BudgetPaymentModel
	require.NoError(t, db.First(&gotPending, "id = ?", pending.ID).Error)
	require.NoError(t, db.First(&gotPaid, "id = ?", paid.ID).Error)

	require.Equal(t, "canceled", gotPending.Status, "pending link must be cancelled")
	require.Equal(t, entities.ReasonBudgetCancelled, gotPending.StatusReason)
	require.Equal(t, "paid", gotPaid.Status, "a paid link must be left untouched")
}
