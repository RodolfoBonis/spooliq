package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// validateReferences ensures that every preset and filament referenced by a
// budget belongs to the caller's organization AND is of the expected type,
// preventing cross-tenant leakage and type mismatches (e.g. a cost preset passed
// as a machine preset).
//
// Only references explicitly provided by the caller should be passed here: a
// reference that is already stored on an existing budget must NOT be re-validated,
// because the preset it points to may have since been soft-deleted — re-checking
// it would make the budget impossible to edit. A returned error is safe to
// surface to the client as a 400 (bad request).
func (uc *BudgetUseCase) validateReferences(
	ctx context.Context,
	organizationID string,
	machinePresetID, energyPresetID, costPresetID *uuid.UUID,
	items []entities.BudgetItemRequest,
) error {
	typed := []struct {
		id         *uuid.UUID
		presetType string
	}{
		{machinePresetID, "machine"},
		{energyPresetID, "energy"},
		{costPresetID, "cost"},
	}
	for _, p := range typed {
		if p.id == nil {
			continue
		}
		if err := uc.budgetRepository.ValidatePresetInOrg(ctx, *p.id, p.presetType, organizationID); err != nil {
			return err
		}
	}

	for _, item := range items {
		if item.CostPresetID != nil {
			if err := uc.budgetRepository.ValidatePresetInOrg(ctx, *item.CostPresetID, "cost", organizationID); err != nil {
				return err
			}
		}
	}

	if err := uc.budgetRepository.ValidateFilamentsInOrg(ctx, collectFilamentIDs(items), organizationID); err != nil {
		return err
	}

	if err := uc.budgetRepository.ValidateModel3DsInOrg(ctx, collectModel3DIDs(items), organizationID); err != nil {
		return err
	}

	return nil
}
