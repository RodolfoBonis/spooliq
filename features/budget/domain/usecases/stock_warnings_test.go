package usecases

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestBuildBudgetResponse_StockWarnings proves the detail response carries stock
// warnings for a non-completed budget and omits them (empty, no query) for a
// completed budget whose stock was already deducted.
func TestBuildBudgetResponse_StockWarnings(t *testing.T) {
	warning := entities.StockWarning{
		FilamentID:     uuid.New().String(),
		FilamentName:   "PLA Vermelho",
		Color:          "red",
		RequiredGrams:  800,
		AvailableGrams: 200,
	}

	t.Run("draft budget includes warnings", func(t *testing.T) {
		repo := &fakeBudgetRepo{
			budget:        &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Status: entities.StatusDraft},
			stockWarnings: []entities.StockWarning{warning},
		}
		uc := newUseCaseWith(repo)

		resp, err := uc.buildBudgetResponse(context.Background(), repo.budget.ID, "org-a")
		require.NoError(t, err)
		require.Equal(t, 1, repo.getStockWarningsCalls, "detail must query warnings for non-completed budgets")
		require.Len(t, resp.StockWarnings, 1)
		require.Equal(t, warning, resp.StockWarnings[0])
	})

	t.Run("completed budget omits warnings", func(t *testing.T) {
		repo := &fakeBudgetRepo{
			budget:        &entities.BudgetEntity{ID: uuid.New(), OrganizationID: "org-a", Status: entities.StatusCompleted},
			stockWarnings: []entities.StockWarning{warning},
		}
		uc := newUseCaseWith(repo)

		resp, err := uc.buildBudgetResponse(context.Background(), repo.budget.ID, "org-a")
		require.NoError(t, err)
		require.Equal(t, 0, repo.getStockWarningsCalls, "completed budgets must not query warnings")
		require.Empty(t, resp.StockWarnings)
		require.NotNil(t, resp.StockWarnings, "stock_warnings must serialize as [] not null")
	})
}
