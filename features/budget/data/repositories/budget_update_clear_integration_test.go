package repositories_test

import (
	"context"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedClearableBudget creates a company (0% default tax), a filament, a cost preset and
// a one-item budget with every clearable field set. It returns the budget id and the
// cost preset id so the caller can tweak the entity before re-updating.
func seedClearableBudget(t *testing.T, db *gorm.DB, ctx context.Context, repo budgetRepo.BudgetRepository, org string, budget *entities.BudgetEntity) {
	t.Helper()
	now := time.Now()

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 10000.0).Error)

	// Company default tax rate 0% so clearing the budget's 10% tax lowers the total.
	require.NoError(t, db.Exec(`INSERT INTO companies (organization_id, default_tax_rate) VALUES (?, ?)`, org, 0.0).Error)

	require.NoError(t, repo.Create(ctx, budget))

	itemID := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID: itemID, BudgetID: budget.ID, FilamentID: fil, OrganizationID: org,
		Quantity: 1000, Order: 1, ProductName: "A", ProductQuantity: 1,
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org, Quantity: 1000, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))
}

// TestIntegration_UpdateClearsTaxRate_LowersTotal proves that persisting a nil TaxRate
// writes NULL to the column (so the map-based Update does not skip it) and that the
// recalculated total drops to the company default (0%) once the 10% budget tax is cleared.
func TestIntegration_UpdateClearsTaxRate_LowersTotal(t *testing.T) {
	db := openTestDB(t, "budget_clear_tax")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	taxRate := 10.0
	budgetID := uuid.New()
	budget := &entities.BudgetEntity{
		ID:             budgetID,
		OrganizationID: org,
		Name:           "Clear tax",
		CustomerID:     uuid.New(),
		Status:         entities.StatusDraft,
		TaxRate:        &taxRate,
		OwnerUserID:    "user-1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	seedClearableBudget(t, db, ctx, repo, org, budget)

	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))
	var total1 int64
	var applied1 float64
	require.NoError(t, db.Raw(`SELECT total_cost, tax_rate_applied FROM budgets WHERE id = ?`, budgetID).Row().Scan(&total1, &applied1))
	require.Equal(t, 10.0, applied1, "budget tax rate applied before clear")
	require.Greater(t, total1, int64(0))

	// Clear the budget-level tax rate and persist.
	budget.TaxRate = nil
	budget.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, budget))

	// The column must be NULL, not left at its previous value.
	var taxIsNull bool
	require.NoError(t, db.Raw(`SELECT tax_rate IS NULL FROM budgets WHERE id = ?`, budgetID).Row().Scan(&taxIsNull))
	require.True(t, taxIsNull, "tax_rate must persist as NULL after clear")

	// Recalculate: the company default (0%) now applies, lowering the total.
	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))
	var total2 int64
	var applied2 float64
	require.NoError(t, db.Raw(`SELECT total_cost, tax_rate_applied FROM budgets WHERE id = ?`, budgetID).Row().Scan(&total2, &applied2))
	require.Equal(t, 0.0, applied2, "company default (0%) applies after clearing budget tax")
	require.Less(t, total2, total1, "clearing a 10%% tax (0%% company default) must lower the total")
}

// TestIntegration_UpdateClearsNullableFields_PersistsNULL proves that clearing the
// discount pair, the shipping override and valid_until all persist NULL through the
// map-based repository Update, and that costs recompute afterwards.
func TestIntegration_UpdateClearsNullableFields_PersistsNULL(t *testing.T) {
	db := openTestDB(t, "budget_clear_all")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	discType := entities.DiscountTypePercent
	discVal := 10.0
	var ship int64 = 500
	taxRate := 10.0
	vu := time.Date(2026, 6, 1, 23, 59, 59, 0, time.UTC)
	budgetID := uuid.New()
	budget := &entities.BudgetEntity{
		ID:               budgetID,
		OrganizationID:   org,
		Name:             "Clear all",
		CustomerID:       uuid.New(),
		Status:           entities.StatusDraft,
		DiscountType:     &discType,
		DiscountValue:    &discVal,
		IncludeShipping:  true,
		ShippingOverride: &ship,
		TaxRate:          &taxRate,
		ValidUntil:       &vu,
		OwnerUserID:      "user-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	seedClearableBudget(t, db, ctx, repo, org, budget)

	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))
	var total1 int64
	require.NoError(t, db.Raw(`SELECT total_cost FROM budgets WHERE id = ?`, budgetID).Row().Scan(&total1))
	require.Greater(t, total1, int64(0))

	// Clear every nullable field and persist.
	budget.DiscountType = nil
	budget.DiscountValue = nil
	budget.ShippingOverride = nil
	budget.TaxRate = nil
	budget.ValidUntil = nil
	budget.UpdatedAt = time.Now()
	require.NoError(t, repo.Update(ctx, budget))

	var row struct {
		DiscountTypeNull  bool
		DiscountValueNull bool
		ShippingNull      bool
		TaxNull           bool
		ValidUntilNull    bool
	}
	require.NoError(t, db.Raw(`SELECT
		discount_type IS NULL  AS discount_type_null,
		discount_value IS NULL AS discount_value_null,
		shipping_override IS NULL AS shipping_null,
		tax_rate IS NULL       AS tax_null,
		valid_until IS NULL    AS valid_until_null
		FROM budgets WHERE id = ?`, budgetID).Scan(&row).Error)
	require.True(t, row.DiscountTypeNull, "discount_type must be NULL")
	require.True(t, row.DiscountValueNull, "discount_value must be NULL")
	require.True(t, row.ShippingNull, "shipping_override must be NULL")
	require.True(t, row.TaxNull, "tax_rate must be NULL")
	require.True(t, row.ValidUntilNull, "valid_until must be NULL")

	// Costs recompute (the update is draft-only and recalculates in the handler).
	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))
	var total2 int64
	require.NoError(t, db.Raw(`SELECT total_cost FROM budgets WHERE id = ?`, budgetID).Row().Scan(&total2))
	require.NotEqual(t, total1, total2, "recalculated total must change after clearing discount/shipping/tax")
}
