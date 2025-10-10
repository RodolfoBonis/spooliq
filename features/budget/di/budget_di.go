package di

import (
	"github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"go.uber.org/fx"
)

// Module exports the budget feature's dependency injection module
var Module = fx.Module(
	"budget",
	fx.Provide(
		repositories.NewBudgetRepository,
		usecases.NewBudgetUseCase,
	),
)
