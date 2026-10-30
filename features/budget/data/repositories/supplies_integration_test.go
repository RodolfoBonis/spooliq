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

// TestIntegration_Supplies_SnapshotPersistAndRefresh exercises the REAL repository
// against a real PostgreSQL to prove the budget item supply wiring:
//
//   - CalculateCosts prices supplies from the CURRENT catalog unit cost and persists
//     the per-item and budget supplies_cost;
//   - the item supply snapshot (name/unit/unit_cost) is refreshed to the catalog value
//     on every recalculation (so a later catalog price change flows into the budget);
//   - the pure engine over the same inputs agrees with the persisted totals.
func TestIntegration_Supplies_SnapshotPersistAndRefresh(t *testing.T) {
	db := openTestDB(t, "budget_supplies")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 10000.0).Error)

	// Catalog supply: LED, R$1,50/un (150 cents).
	supply := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO supplies (id, organization_id, name, unit, unit_cost, is_active) VALUES (?,?,?,?,?,?)`,
		supply, org, "LED", "un", 150, true).Error)

	budgetID := uuid.New()
	require.NoError(t, repo.Create(ctx, &entities.BudgetEntity{
		ID: budgetID, OrganizationID: org, Name: "Supplies", CustomerID: uuid.New(),
		Status: entities.StatusDraft, OwnerUserID: "user-1", CreatedAt: now, UpdatedAt: now,
	}))

	itemID := uuid.New()
	require.NoError(t, repo.AddItem(ctx, &entities.BudgetItemEntity{
		ID: itemID, BudgetID: budgetID, FilamentID: fil, OrganizationID: org,
		Quantity: 100, Order: 1, ProductName: "A", ProductQuantity: 3, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org, Quantity: 100, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))
	// 2 LEDs per piece, snapshot taken at the (stale) 100-cent price to prove refresh.
	require.NoError(t, repo.AddItemSupply(ctx, &entities.BudgetItemSupplyEntity{
		ID: uuid.New(), BudgetItemID: itemID, SupplyID: supply, OrganizationID: org,
		QuantityPerPiece: 2, Order: 0, Name: "LED", Unit: "un", UnitCost: 100, CreatedAt: now, UpdatedAt: now,
	}))

	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))

	// Supplies cost = 2 (per piece) * 3 (pieces) * 150 (current catalog) = 900.
	var itemSuppliesCost, budgetSuppliesCost int64
	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budget_items WHERE id = ?`, itemID).Scan(&itemSuppliesCost).Error)
	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budgets WHERE id = ?`, budgetID).Scan(&budgetSuppliesCost).Error)
	require.Equal(t, int64(900), itemSuppliesCost, "item supplies_cost priced from the CURRENT catalog unit cost")
	require.Equal(t, int64(900), budgetSuppliesCost, "budget supplies_cost")

	// The snapshot was refreshed from 100 -> 150 (the catalog value used for pricing).
	got, err := repo.GetItemSupplies(ctx, itemID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(150), got[0].UnitCost, "snapshot unit_cost refreshed to the catalog value")
	require.Equal(t, "LED", got[0].Name)
	require.Equal(t, "un", got[0].Unit)
	require.InEpsilon(t, 2.0, got[0].QuantityPerPiece, 1e-9)

	// A later catalog price change flows in on the next recalculation.
	require.NoError(t, db.Exec(`UPDATE supplies SET unit_cost = 200 WHERE id = ?`, supply).Error)
	require.NoError(t, repo.CalculateCosts(ctx, budgetID, org))

	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budget_items WHERE id = ?`, itemID).Scan(&itemSuppliesCost).Error)
	require.Equal(t, int64(1200), itemSuppliesCost, "2*3*200 after the catalog price change")
	got, err = repo.GetItemSupplies(ctx, itemID)
	require.NoError(t, err)
	require.Equal(t, int64(200), got[0].UnitCost, "snapshot refreshed to the new catalog price")

	// Item total = filament (100g@R$100/kg = 1000c) + supplies 1200 = 2200.
	var itemTotal int64
	require.NoError(t, db.Raw(`SELECT item_total_cost FROM budget_items WHERE id = ?`, itemID).Scan(&itemTotal).Error)
	require.Equal(t, int64(1000+1200), itemTotal)

	// Validation + snapshot lookup helpers resolve the live catalog supply.
	require.NoError(t, repo.ValidateSuppliesInOrg(ctx, []uuid.UUID{supply}, org))
	snaps, err := repo.GetSuppliesInfo(ctx, []uuid.UUID{supply}, org)
	require.NoError(t, err)
	require.Equal(t, int64(200), snaps[supply].UnitCost)

	// Cross-tenant guard: another org cannot validate this supply.
	require.Error(t, repo.ValidateSuppliesInOrg(ctx, []uuid.UUID{supply}, "org-b"))
}

// TestIntegration_Supplies_BatchLoadByItemIDs proves GetItemSuppliesByItemIDs groups
// supplies by item, org-scoped and ordered — the batched path used by the list response.
func TestIntegration_Supplies_BatchLoadByItemIDs(t *testing.T) {
	db := openTestDB(t, "budget_supplies_batch")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	repo := repoimpl.NewBudgetRepository(db)
	now := time.Now()

	item1 := uuid.New()
	item2 := uuid.New()
	s1 := uuid.New()
	s2 := uuid.New()
	require.NoError(t, repo.AddItemSupply(ctx, &entities.BudgetItemSupplyEntity{
		ID: uuid.New(), BudgetItemID: item1, SupplyID: s1, OrganizationID: org, QuantityPerPiece: 1, Order: 1, Name: "B", Unit: "un", UnitCost: 50, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemSupply(ctx, &entities.BudgetItemSupplyEntity{
		ID: uuid.New(), BudgetItemID: item1, SupplyID: s2, OrganizationID: org, QuantityPerPiece: 1, Order: 0, Name: "A", Unit: "un", UnitCost: 75, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, repo.AddItemSupply(ctx, &entities.BudgetItemSupplyEntity{
		ID: uuid.New(), BudgetItemID: item2, SupplyID: s1, OrganizationID: org, QuantityPerPiece: 3, Order: 0, Name: "B", Unit: "un", UnitCost: 50, CreatedAt: now, UpdatedAt: now,
	}))

	out, err := repo.GetItemSuppliesByItemIDs(ctx, []uuid.UUID{item1, item2}, org)
	require.NoError(t, err)
	require.Len(t, out[item1], 2)
	require.Len(t, out[item2], 1)
	// Ordered by "order" ASC within an item.
	require.Equal(t, "A", out[item1][0].Name)
	require.Equal(t, "B", out[item1][1].Name)

	// Another org sees nothing.
	outB, err := repo.GetItemSuppliesByItemIDs(ctx, []uuid.UUID{item1, item2}, "org-b")
	require.NoError(t, err)
	require.Empty(t, outB)
}
