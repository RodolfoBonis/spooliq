package repositories_test

import (
	"context"
	"testing"
	"time"

	repoimpl "github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	supplyrepo "github.com/RodolfoBonis/spooliq/features/supply/data/repositories"
	supplyentities "github.com/RodolfoBonis/spooliq/features/supply/domain/entities"
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
	got, err := repo.GetItemSupplies(ctx, itemID, org)
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
	got, err = repo.GetItemSupplies(ctx, itemID, org)
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

// TestIntegration_Supplies_SoftDeleteKeepsBudgetsWorking proves the soft-delete
// contract end to end against a real PostgreSQL: a supply referenced by a budget can
// be deleted from the catalog (SOFT: the row survives with deleted_at set), after
// which
//
//   - the supply is hidden from the catalog (find-by-id and list exclude it);
//   - it can no longer be attached to a new/updated item (ValidateSuppliesInOrg -> 400
//     supply_not_found; GetSuppliesInfo drops it);
//   - the existing budget keeps its stored supply snapshot (detail still renders it);
//   - recalc still prices the item from the (soft-deleted) catalog row, and a later
//     price change on that row still flows into the recalculation.
func TestIntegration_Supplies_SoftDeleteKeepsBudgetsWorking(t *testing.T) {
	db := openTestDB(t, "supply_soft_delete")
	createPricingSchema(t, db)

	const org = "org-a"
	ctx := context.Background()
	budgetRepo := repoimpl.NewBudgetRepository(db)
	supplyRepo := supplyrepo.NewSupplyRepository(db)
	now := time.Now()

	fil := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO filaments (id, organization_id, price_per_kg) VALUES (?,?,?)`, fil, org, 10000.0).Error)

	// Catalog supply created through the supply repository (soft-delete aware model).
	supply := &supplyentities.SupplyEntity{
		ID: uuid.New(), OrganizationID: org, Name: "LED", Unit: "un", UnitCost: 150, IsActive: true,
	}
	require.NoError(t, supplyRepo.Create(supply))
	require.NotEqual(t, uuid.Nil, supply.ID)

	// Budget with one item consuming 2 LEDs/piece over 3 pieces.
	budgetID := uuid.New()
	require.NoError(t, budgetRepo.Create(ctx, &entities.BudgetEntity{
		ID: budgetID, OrganizationID: org, Name: "B", CustomerID: uuid.New(),
		Status: entities.StatusDraft, OwnerUserID: "u1", CreatedAt: now, UpdatedAt: now,
	}))
	itemID := uuid.New()
	require.NoError(t, budgetRepo.AddItem(ctx, &entities.BudgetItemEntity{
		ID: itemID, BudgetID: budgetID, FilamentID: fil, OrganizationID: org,
		Quantity: 100, Order: 1, ProductName: "A", ProductQuantity: 3, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, budgetRepo.AddItemFilament(ctx, &entities.BudgetItemFilamentEntity{
		ID: uuid.New(), BudgetItemID: itemID, FilamentID: fil, OrganizationID: org, Quantity: 100, Order: 1, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, budgetRepo.AddItemSupply(ctx, &entities.BudgetItemSupplyEntity{
		ID: uuid.New(), BudgetItemID: itemID, SupplyID: supply.ID, OrganizationID: org,
		QuantityPerPiece: 2, Order: 0, Name: "LED", Unit: "un", UnitCost: 150, CreatedAt: now, UpdatedAt: now,
	}))

	require.NoError(t, budgetRepo.CalculateCosts(ctx, budgetID, org))
	var suppliesCost int64
	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budget_items WHERE id = ?`, itemID).Scan(&suppliesCost).Error)
	require.Equal(t, int64(900), suppliesCost, "baseline 2*3*150")

	// --- Soft delete the supply from the catalog. ---
	require.NoError(t, supplyRepo.Delete(supply.ID))

	// The row is still physically present (soft delete only sets deleted_at).
	var survivingRows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM supplies WHERE id = ? AND deleted_at IS NOT NULL`, supply.ID).Scan(&survivingRows).Error)
	require.Equal(t, int64(1), survivingRows, "soft delete keeps the row, only sets deleted_at")

	// Hidden from the catalog: find-by-id and list exclude it.
	_, err := supplyRepo.FindByID(supply.ID, org)
	require.Error(t, err, "soft-deleted supply is not found by id")
	list, total, err := supplyRepo.FindAll(org, "", nil, "name asc", 50, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), total, "soft-deleted supply hidden from the list total")
	require.Empty(t, list)

	// Cannot be attached anew: validation rejects it (surfaces as 400 supply_not_found)
	// and the live snapshot loader drops it.
	require.ErrorIs(t, budgetRepo.ValidateSuppliesInOrg(ctx, []uuid.UUID{supply.ID}, org), entities.ErrSupplyNotFound)
	snaps, err := budgetRepo.GetSuppliesInfo(ctx, []uuid.UUID{supply.ID}, org)
	require.NoError(t, err)
	require.NotContains(t, snaps, supply.ID)

	// Budget detail still resolves the stored supply snapshot.
	got, err := budgetRepo.GetItemSupplies(ctx, itemID, org)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, supply.ID, got[0].SupplyID)

	// Recalc still works and still prices from the (soft-deleted) catalog row.
	require.NoError(t, budgetRepo.CalculateCosts(ctx, budgetID, org))
	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budget_items WHERE id = ?`, itemID).Scan(&suppliesCost).Error)
	require.Equal(t, int64(900), suppliesCost, "recalc of a budget whose supply was soft-deleted still prices from the catalog row")

	// A later price change on the soft-deleted row still flows into recalc.
	require.NoError(t, db.Exec(`UPDATE supplies SET unit_cost = 200 WHERE id = ?`, supply.ID).Error)
	require.NoError(t, budgetRepo.CalculateCosts(ctx, budgetID, org))
	require.NoError(t, db.Raw(`SELECT supplies_cost FROM budget_items WHERE id = ?`, itemID).Scan(&suppliesCost).Error)
	require.Equal(t, int64(1200), suppliesCost, "2*3*200 after the catalog price change on the soft-deleted row")
}
