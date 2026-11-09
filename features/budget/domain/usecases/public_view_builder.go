package usecases

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
)

// PublicViewBuilder builds the sanitized public budget view by share token. It is
// implemented by PublicBudgetUseCase and consumed by the payments feature so the
// public confirm endpoint can return the SAME body as GET /public/budgets/:token
// (payment block included) without the payments package reimplementing it.
type PublicViewBuilder interface {
	BuildPublicViewByToken(ctx context.Context, token string) (*entities.PublicBudgetView, error)
}

// BuildPublicViewByToken looks the budget up by token and assembles its sanitized
// public view (customer, company, items, totals and the payment block). It performs
// no writes. Callers that need the "under review" guard (draft) must check it
// separately; this builder renders whatever budget the token resolves to.
func (uc *PublicBudgetUseCase) BuildPublicViewByToken(ctx context.Context, token string) (*entities.PublicBudgetView, error) {
	budget, err := uc.budgetRepository.FindByPublicToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return uc.buildPublicView(ctx, budget)
}
