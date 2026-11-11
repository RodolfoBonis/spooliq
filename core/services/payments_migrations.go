package services

import (
	"fmt"

	paymentModels "github.com/RodolfoBonis/spooliq/features/payments/data/models"
)

// RunPaymentsMigrations migrates the payments tables (InfinitePay integration,
// budget payments, webhook events) and adds their FK constraints and indexes
// idempotently. It runs AFTER RunMigrations so the budgets/companies tables already
// exist. Everything here is additive: the feature stays inert until an organization
// connects a provider, and all statements are safe to re-run.
//
// The budgets.deposit_percent / payment_status / amount_paid_cents /
// amount_refund_due_cents / paid_at columns are added by RunMigrations' AutoMigrate of
// the budget model.
func RunPaymentsMigrations() {
	if Connector == nil {
		fmt.Println("Warning: payments migrations skipped (no DB connection)")
		return
	}

	// Backfill any NULL status_reason/review_reason BEFORE AutoMigrate adds the
	// NOT NULL constraint (a pre-existing budget_payments from an earlier build of this
	// change may have NULLs that would make the ALTER fail).
	backfillPaymentNulls()

	// ========================================
	// LEVEL 6: Payments (depends on Companies and Budgets)
	// ========================================
	if err := Connector.AutoMigrate(
		&paymentModels.PaymentIntegrationModel{},
		&paymentModels.BudgetPaymentModel{},
		&paymentModels.PaymentWebhookEventModel{},
	); err != nil {
		panic(fmt.Sprintf("ERROR DURING PAYMENTS MIGRATION: %s", err.Error()))
	}

	// organization_id FK constraints (companies ON DELETE RESTRICT), idempotent.
	paymentsOrgFKTables := []string{"payment_integrations", "budget_payments", "payment_webhook_events"}
	for _, table := range paymentsOrgFKTables {
		var exists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = ?)", table).Scan(&exists)
		if !exists {
			continue
		}
		var fkExists bool
		Connector.Raw(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema = current_schema() AND table_name = ? AND constraint_name LIKE '%organization%'
			)
		`, table).Scan(&fkExists)
		if !fkExists {
			sql := fmt.Sprintf(`
				ALTER TABLE %s
				ADD CONSTRAINT fk_%s_organization
				FOREIGN KEY (organization_id)
				REFERENCES companies(organization_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT
			`, table, table)
			if err := Connector.Exec(sql).Error; err != nil {
				fmt.Printf("Warning: FK constraint for %s.organization_id failed: %v\n", table, err)
			} else {
				fmt.Printf("Added FK constraint for %s.organization_id\n", table)
			}
		}
	}

	// budget_payments.budget_id FK -> budgets(id) ON DELETE RESTRICT.
	{
		var budgetPaymentsExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'budget_payments')").Scan(&budgetPaymentsExists)
		if budgetPaymentsExists {
			var fkExists bool
			Connector.Raw(`
				SELECT EXISTS(
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_schema = current_schema() AND table_name = 'budget_payments' AND constraint_name = 'fk_budget_payments_budget'
				)
			`).Scan(&fkExists)
			if !fkExists {
				sql := `
					ALTER TABLE budget_payments
					ADD CONSTRAINT fk_budget_payments_budget
					FOREIGN KEY (budget_id)
					REFERENCES budgets(id)
					ON UPDATE CASCADE
					ON DELETE RESTRICT
				`
				if err := Connector.Exec(sql).Error; err != nil {
					fmt.Printf("Warning: FK constraint for budget_payments.budget_id failed: %v\n", err)
				} else {
					fmt.Println("Added FK constraint for budget_payments.budget_id")
				}
			}
		}
	}

	// ========================================
	// PAYMENTS INDEXES (idempotent). The CRITICAL unique indexes enforce money-safety
	// invariants; failing to create them must FAIL STARTUP rather than run without them.
	// ========================================
	paymentIndexes := []struct {
		name     string
		sql      string
		critical bool
	}{
		{
			"uq_payment_integrations_org_provider",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_payment_integrations_org_provider ON payment_integrations(organization_id, provider)",
			true,
		},
		{
			"uq_budget_payments_order_nsu",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_budget_payments_order_nsu ON budget_payments(order_nsu)",
			true,
		},
		{
			"uq_budget_payments_transaction_nsu",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_budget_payments_transaction_nsu ON budget_payments(transaction_nsu) WHERE transaction_nsu IS NOT NULL",
			true,
		},
		{
			// The anti-race guard: at most one pending link per (budget, kind).
			"uq_budget_payments_pending",
			"CREATE UNIQUE INDEX IF NOT EXISTS uq_budget_payments_pending ON budget_payments(budget_id, kind) WHERE status = 'pending'",
			true,
		},
		{
			// Reconcile job sweeps by status + age. Note: no GORM index tag shadows this.
			"idx_budget_payments_status_created",
			"CREATE INDEX IF NOT EXISTS idx_budget_payments_status_created ON budget_payments(status, created_at)",
			false,
		},
		{
			"idx_payment_webhook_events_unprocessed",
			"CREATE INDEX IF NOT EXISTS idx_payment_webhook_events_unprocessed ON payment_webhook_events(received_at) WHERE processed_at IS NULL",
			false,
		},
	}
	for _, idx := range paymentIndexes {
		if err := Connector.Exec(idx.sql).Error; err != nil {
			if idx.critical {
				panic(fmt.Sprintf("ERROR: critical payments index %s failed to create: %s", idx.name, err.Error()))
			}
			fmt.Printf("Warning: Index %s creation failed: %v\n", idx.name, err)
		} else {
			fmt.Printf("Created index %s\n", idx.name)
		}
	}

	fmt.Println("Payments migrations setup completed")
}

// backfillPaymentNulls sets any NULL status_reason/review_reason to ” on an existing
// budget_payments table so the subsequent AutoMigrate can add NOT NULL safely.
func backfillPaymentNulls() {
	var tableExists bool
	Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema = current_schema() AND table_name = 'budget_payments')").Scan(&tableExists)
	if !tableExists {
		return
	}
	for _, col := range []string{"status_reason", "review_reason"} {
		var colExists bool
		Connector.Raw("SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'budget_payments' AND column_name = ?)", col).Scan(&colExists)
		if !colExists {
			continue
		}
		if err := Connector.Exec(fmt.Sprintf("UPDATE budget_payments SET %s = '' WHERE %s IS NULL", col, col)).Error; err != nil {
			fmt.Printf("Warning: backfill of budget_payments.%s failed: %v\n", col, err)
		}
	}
}
