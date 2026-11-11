package usecases

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/google/uuid"
)

// attachRequestSupplies resolves the catalog snapshot (name/unit/unit_cost) for
// every supply referenced by the request items and attaches the built supply
// entities to each item in `built`. `built` and `reqs` are positionally aligned
// (buildBudgetItems preserves request order). Items whose request Supplies slice is
// non-nil get those supplies; a nil slice yields no supplies (on CREATE that simply
// means "no supplies"; the UPDATE preserve-on-absent rule is handled separately).
func (uc *BudgetUseCase) attachRequestSupplies(ctx context.Context, organizationID string, built []builtBudgetItem, reqs []entities.BudgetItemRequest) error {
	snapshots, err := uc.budgetRepository.GetSuppliesInfo(ctx, collectSupplyIDs(reqs), organizationID)
	if err != nil {
		return err
	}
	now := time.Now()
	for i := range built {
		built[i].Supplies = buildItemSupplies(built[i].Item.ID, organizationID, reqs[i].Supplies, snapshots, now)
	}
	return nil
}

// attachUpdateSupplies attaches supplies to the rebuilt items following the web
// compatibility rule: a request item whose `supplies` field is ABSENT (nil slice)
// preserves the PREVIOUS item's supplies — items are recreated on update, so the
// previous supplies are matched by the item's POSITION in the ordered item list.
// An explicit empty array clears the item's supplies; a non-empty array replaces
// them with freshly snapshotted catalog values. `built` and `reqs` are positionally
// aligned (buildBudgetItems preserves request order).
//
// It MUST run inside the update transaction (receiving the transaction-bound repo)
// and AFTER the draft-guarded budget update, so the preserve-by-position read of the
// previous items/supplies sees the same committed state the item replacement will
// delete — closing the TOCTOU window a pre-transaction read would leave open.
func (uc *BudgetUseCase) attachUpdateSupplies(ctx context.Context, repo budgetRepo.BudgetRepository, organizationID string, budgetID uuid.UUID, built []builtBudgetItem, reqs []entities.BudgetItemRequest) error {
	// Previous items (ordered by "order") and their supplies, for the preserve rule.
	prevItems, err := repo.GetItems(ctx, budgetID)
	if err != nil {
		return err
	}
	prevItemIDs := make([]uuid.UUID, len(prevItems))
	for i, it := range prevItems {
		prevItemIDs[i] = it.ID
	}
	prevSuppliesByItem, err := repo.GetItemSuppliesByItemIDs(ctx, prevItemIDs, organizationID)
	if err != nil {
		return err
	}

	snapshots, err := repo.GetSuppliesInfo(ctx, collectSupplyIDs(reqs), organizationID)
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range built {
		if reqs[i].Supplies == nil {
			// Absent: preserve the previous item's supplies matched by position.
			var prev []*entities.BudgetItemSupplyEntity
			if i < len(prevItems) {
				prev = prevSuppliesByItem[prevItems[i].ID]
			}
			built[i].Supplies = copyItemSupplies(built[i].Item.ID, organizationID, prev, now)
		} else {
			built[i].Supplies = buildItemSupplies(built[i].Item.ID, organizationID, reqs[i].Supplies, snapshots, now)
		}
	}
	return nil
}
