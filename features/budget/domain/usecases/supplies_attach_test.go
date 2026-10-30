package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// preserveRepo embeds the package's fakeBudgetRepo and overrides only the reads
// attachUpdateSupplies performs, so the positional preserve rule can be exercised
// without a database.
type preserveRepo struct {
	*fakeBudgetRepo
	prevItems    []*entities.BudgetItemEntity
	prevSupplies map[uuid.UUID][]*entities.BudgetItemSupplyEntity
	snapshots    map[uuid.UUID]entities.SupplySnapshot
}

func (r *preserveRepo) GetItems(_ context.Context, _ uuid.UUID) ([]*entities.BudgetItemEntity, error) {
	return r.prevItems, nil
}

func (r *preserveRepo) GetItemSuppliesByItemIDs(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID][]*entities.BudgetItemSupplyEntity, error) {
	return r.prevSupplies, nil
}

func (r *preserveRepo) GetSuppliesInfo(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID]entities.SupplySnapshot, error) {
	return r.snapshots, nil
}

// TestAttachUpdateSupplies_PreserveClearReplace verifies the three web-compatibility
// cases of the update supply rule, matching previous items by POSITION/order:
//
//   - item 0: `supplies` ABSENT (nil) -> preserve the previous item-0 supplies;
//   - item 1: explicit [] -> clear;
//   - item 2: non-empty -> replace with freshly snapshotted catalog values.
func TestAttachUpdateSupplies_PreserveClearReplace(t *testing.T) {
	const org = "org-a"

	prevItem0 := uuid.New()
	prevItem1 := uuid.New()
	prevItem2 := uuid.New()
	preservedSupplyID := uuid.New()
	newSupplyID := uuid.New()

	repo := &preserveRepo{
		fakeBudgetRepo: &fakeBudgetRepo{},
		prevItems: []*entities.BudgetItemEntity{
			{ID: prevItem0, Order: 1},
			{ID: prevItem1, Order: 2},
			{ID: prevItem2, Order: 3},
		},
		prevSupplies: map[uuid.UUID][]*entities.BudgetItemSupplyEntity{
			prevItem0: {{
				ID: uuid.New(), BudgetItemID: prevItem0, SupplyID: preservedSupplyID, OrganizationID: org,
				QuantityPerPiece: 4, Order: 0, Name: "Old LED", Unit: "un", UnitCost: 111,
			}},
			// prevItem1 also had a supply; the explicit [] must clear it anyway.
			prevItem1: {{
				ID: uuid.New(), BudgetItemID: prevItem1, SupplyID: preservedSupplyID, OrganizationID: org,
				QuantityPerPiece: 9, Order: 0, Name: "Should be cleared", Unit: "un", UnitCost: 999,
			}},
		},
		snapshots: map[uuid.UUID]entities.SupplySnapshot{
			newSupplyID: {Name: "New Keyring", Unit: "un", UnitCost: 250},
		},
	}

	uc := &BudgetUseCase{budgetRepository: repo}

	reqs := []entities.BudgetItemRequest{
		// item 0: supplies ABSENT (nil slice) -> preserve.
		{ProductName: "P0", ProductQuantity: 1, Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: uuid.New(), Quantity: 10, Order: 1}}},
		// item 1: explicit empty -> clear.
		{ProductName: "P1", ProductQuantity: 1, Supplies: []entities.BudgetItemSupplyRequest{}, Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: uuid.New(), Quantity: 10, Order: 1}}},
		// item 2: non-empty -> replace.
		{ProductName: "P2", ProductQuantity: 1, Supplies: []entities.BudgetItemSupplyRequest{{SupplyID: newSupplyID, QuantityPerPiece: 2}}, Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: uuid.New(), Quantity: 10, Order: 1}}},
	}

	built := buildBudgetItems(uuid.New(), org, reqs)
	if err := uc.attachUpdateSupplies(context.Background(), repo, org, uuid.New(), built, reqs); err != nil {
		t.Fatalf("attachUpdateSupplies: %v", err)
	}

	// item 0: preserved from previous item-0 (snapshot copied, new row ID, new item ID).
	if len(built[0].Supplies) != 1 {
		t.Fatalf("item 0 supplies len = %d, want 1 (preserved)", len(built[0].Supplies))
	}
	p := built[0].Supplies[0]
	if p.SupplyID != preservedSupplyID || p.Name != "Old LED" || p.UnitCost != 111 || p.QuantityPerPiece != 4 {
		t.Errorf("item 0 preserved supply = %+v, want the previous item-0 supply", p)
	}
	if p.BudgetItemID != built[0].Item.ID {
		t.Errorf("preserved supply must be re-parented to the new item ID")
	}

	// item 1: explicit [] cleared it despite the previous item having a supply.
	if len(built[1].Supplies) != 0 {
		t.Errorf("item 1 supplies len = %d, want 0 (explicit [] clears)", len(built[1].Supplies))
	}

	// item 2: replaced with the freshly snapshotted catalog supply.
	if len(built[2].Supplies) != 1 {
		t.Fatalf("item 2 supplies len = %d, want 1 (replace)", len(built[2].Supplies))
	}
	r := built[2].Supplies[0]
	if r.SupplyID != newSupplyID || r.Name != "New Keyring" || r.UnitCost != 250 || r.QuantityPerPiece != 2 {
		t.Errorf("item 2 supply = %+v, want the new catalog snapshot", r)
	}
}

// TestCopyItemSupplies_DuplicatesWithNewIDs verifies the duplicate/preserve copy
// helper re-parents supplies onto a new item ID with fresh row IDs while keeping the
// supply_id, quantity, order and snapshot.
func TestCopyItemSupplies_DuplicatesWithNewIDs(t *testing.T) {
	const org = "org-a"
	srcItem := uuid.New()
	newItem := uuid.New()
	supplyID := uuid.New()
	origRowID := uuid.New()

	src := []*entities.BudgetItemSupplyEntity{{
		ID: origRowID, BudgetItemID: srcItem, SupplyID: supplyID, OrganizationID: org,
		QuantityPerPiece: 3, Order: 1, Name: "LED", Unit: "un", UnitCost: 150,
	}}

	out := copyItemSupplies(newItem, org, src, time.Now())
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	c := out[0]
	if c.ID == origRowID {
		t.Errorf("copy must get a new row ID")
	}
	if c.BudgetItemID != newItem {
		t.Errorf("copy must be parented to the new item ID")
	}
	if c.SupplyID != supplyID || c.QuantityPerPiece != 3 || c.Order != 1 || c.Name != "LED" || c.Unit != "un" || c.UnitCost != 150 {
		t.Errorf("copy lost data: %+v", c)
	}
}
