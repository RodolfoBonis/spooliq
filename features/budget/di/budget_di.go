package di

import (
	"github.com/RodolfoBonis/spooliq/features/budget/data/repositories"
	budgetRepos "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	presetRepos "github.com/RodolfoBonis/spooliq/features/preset/domain/repositories"
	profileRepos "github.com/RodolfoBonis/spooliq/features/profile/domain/repositories"
	"gorm.io/gorm"

	"go.uber.org/fx"
)

// Module exports the budget feature's dependency injection module
var Module = fx.Module(
	"budget",
	fx.Provide(
		// The repository embeds the org's payment fee into draft pricing via the
		// PaymentFeeProvider port (implemented by the payments feature). The fee
		// provider depends only on the payment repository, so no FX cycle forms.
		func(db *gorm.DB, feeProvider budgetRepos.PaymentFeeProvider) budgetRepos.BudgetRepository {
			return repositories.NewBudgetRepositoryWithFeeProvider(db, feeProvider)
		},
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
		usecases.NewPublicBudgetUseCase,
	),
	// Background job: periodically expire overdue sent budgets.
	fx.Invoke(RegisterExpiryJob),
)
