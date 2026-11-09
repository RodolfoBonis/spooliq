package repositories_test

import (
	"context"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestIntegration_UpdatePersistsDepositPercent proves the budget repository Update
// writes AND clears deposit_percent (it is part of the generic Update column map).
func TestIntegration_UpdatePersistsDepositPercent(t *testing.T) {
	db := openTestDB(t, "budget_deposit")
	createPricingSchema(t, db)
	repo := repoimpl.NewBudgetRepository(db)
	ctx := context.Background()

	org := "org-deposit"
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO budgets (id, organization_id, name, customer_id, status, owner_user_id) VALUES (?,?,?,?,?,?)`,
		id, org, "Vaso", uuid.New(), "draft", "user-1",
	).Error)

	fifty := 50.0
	base := &entities.BudgetEntity{
		ID: id, OrganizationID: org, Name: "Vaso", CustomerID: uuid.New(),
		Status: entities.StatusDraft, DepositPercent: &fifty, UpdatedAt: time.Now(),
	}
	require.NoError(t, repo.Update(ctx, base))

	var got *float64
	require.NoError(t, db.Raw(`SELECT deposit_percent FROM budgets WHERE id = ?`, id).Scan(&got).Error)
	require.NotNil(t, got)
	require.Equal(t, 50.0, *got)

	// Clearing it (nil) must persist NULL.
	base.DepositPercent = nil
	require.NoError(t, repo.Update(ctx, base))
	var cleared *float64
	require.NoError(t, db.Raw(`SELECT deposit_percent FROM budgets WHERE id = ?`, id).Scan(&cleared).Error)
	require.Nil(t, cleared)
}
