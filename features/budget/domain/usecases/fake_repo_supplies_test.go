package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// Supply-related BudgetRepository methods for fakeBudgetRepo. They default to
// no-op/empty behaviour; tests that exercise supplies override the backing fields.

func (f *fakeBudgetRepo) AddItemSupply(_ context.Context, _ *entities.BudgetItemSupplyEntity) error {
	return nil
}

func (f *fakeBudgetRepo) GetItemSupplies(_ context.Context, _ uuid.UUID, _ string) ([]*entities.BudgetItemSupplyEntity, error) {
	return nil, nil
}

func (f *fakeBudgetRepo) GetItemSuppliesByItemIDs(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID][]*entities.BudgetItemSupplyEntity, error) {
	return map[uuid.UUID][]*entities.BudgetItemSupplyEntity{}, nil
}

func (f *fakeBudgetRepo) ValidateSuppliesInOrg(_ context.Context, _ []uuid.UUID, _ string) error {
	return nil
}

func (f *fakeBudgetRepo) GetSuppliesInfo(_ context.Context, _ []uuid.UUID, _ string) (map[uuid.UUID]entities.SupplySnapshot, error) {
	return map[uuid.UUID]entities.SupplySnapshot{}, nil
}
