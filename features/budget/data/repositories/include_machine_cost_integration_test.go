package repositories_test

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/data/models"
	"github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Create_PersistsExplicitIncludeMachineCost guards against GORM omitting
// a zero-valued bool with a default tag from INSERT: an explicit false must be stored as
// false (no machine time charged), and true as true.
func TestIntegration_Create_PersistsExplicitIncludeMachineCost(t *testing.T) {
	db := openTestDB(t, "budget_machine_flag")
	require.NoError(t, db.AutoMigrate(&models.BudgetModel{}))
	repo := repositories.NewBudgetRepository(db)

	for _, want := range []bool{false, true} {
		budget := &entities.BudgetEntity{
			ID:                 uuid.New(),
			OrganizationID:     "org-machine-flag",
			Name:               "machine flag",
			CustomerID:         uuid.New(),
			Status:             entities.StatusDraft,
			IncludeMachineCost: want,
		}
		require.NoError(t, repo.Create(context.Background(), budget))

		var stored bool
		require.NoError(t, db.Raw(`SELECT include_machine_cost FROM budgets WHERE id = ?`, budget.ID).Scan(&stored).Error)
		require.Equal(t, want, stored, "include_machine_cost must be persisted as sent")
	}
}
