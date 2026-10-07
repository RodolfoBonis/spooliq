package usecases

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// Phase 4B interface methods for the shared fakeBudgetRepo used by the use-case unit
// tests. They are intentionally minimal: the Phase 4B flows that need real behavior
// are covered by the dedicated fakes in the public/share tests and by the Postgres
// integration tests.

func (f *fakeBudgetRepo) AllocateQuoteNumber(_ context.Context, _ string) (int, error) {
	return 1, nil
}

func (f *fakeBudgetRepo) FindByPublicToken(_ context.Context, _ string) (*entities.BudgetEntity, error) {
	return nil, entities.ErrBudgetNotFound
}

func (f *fakeBudgetRepo) SetShareToken(_ context.Context, _ uuid.UUID, _ string, _ string, _ time.Time) error {
	return nil
}

func (f *fakeBudgetRepo) RevokeShareToken(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (f *fakeBudgetRepo) SetValidUntil(_ context.Context, _ uuid.UUID, _ string, _ *time.Time) error {
	return nil
}

func (f *fakeBudgetRepo) ExpireOverdue(_ context.Context) (int64, error) {
	return 0, nil
}

func (f *fakeBudgetRepo) RespondToPublicBudget(_ context.Context, _ uuid.UUID, _ entities.BudgetStatus, _, _, _ string, _ *string, _ time.Time) (int64, error) {
	return 1, nil
}

func (f *fakeBudgetRepo) GetCompanyQuoteDefaults(_ context.Context, _ string) (int, *string, error) {
	return 15, nil, nil
}
