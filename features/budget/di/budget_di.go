package di

import (
	"github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"go.uber.org/fx"
)

// Module exports the budget feature's dependency injection module
var Module = fx.Module(
	"budget",
	fx.Provide(
		repositories.NewBudgetRepository,
		// Resolver providers: narrow adapters over the profile and preset
		// repositories, supplied as the budget package's own interfaces so the
		// preset resolver can be wired without importing those features' use cases.
		func(repo profileRepos.PrintProfileRepository) usecases.ProfilePresetProvider {
			return NewProfilePresetProvider(repo)
		},
		func(repo presetRepos.PresetRepository) usecases.DefaultPresetProvider {
			return NewDefaultPresetProvider(repo)
		},
		usecases.NewBudgetUseCase,
	),
)
