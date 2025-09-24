package di

import (
	calculationDI "github.com/RodolfoBonis/spooliq/features/calculation/di"
	filamentsRepos "github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	quotesRepositories "github.com/RodolfoBonis/spooliq/features/quotes/data/repositories"
	quotesServices "github.com/RodolfoBonis/spooliq/features/quotes/domain/services"
	quotesUseCases "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"go.uber.org/fx"
)

// QuotesModule provides dependency injection for quote-related components.
var QuotesModule = fx.Module("quotes",
	// Include calculation service
	calculationDI.Module,

	// Repositories
	fx.Provide(quotesRepositories.NewQuoteRepository),

	// Services - SnapshotService depends on FilamentRepository (provided by FilamentsModule)
	fx.Provide(func(filamentRepo filamentsRepos.FilamentRepository) quotesServices.SnapshotService {
		return quotesServices.NewSnapshotService(filamentRepo)
	}),

	// Services - All Profile Services depend on PresetRepository (provided by PresetsModule at app level)
	fx.Provide(quotesServices.NewEnergyProfileService),
	fx.Provide(quotesServices.NewMachineProfileService),
	fx.Provide(quotesServices.NewCostProfileService),
	fx.Provide(quotesServices.NewMarginProfileService),

	// Use Cases
	fx.Provide(quotesUseCases.NewQuoteUseCase),
)
