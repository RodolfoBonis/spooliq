package di

import (
	calculationDI "github.com/RodolfoBonis/spooliq/features/calculation/di"
	filamentDI "github.com/RodolfoBonis/spooliq/features/filaments/di"
	filamentsRepos "github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	quotesRepositories "github.com/RodolfoBonis/spooliq/features/quotes/data/repositories"
	quotesServices "github.com/RodolfoBonis/spooliq/features/quotes/domain/services"
	quotesUseCases "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"go.uber.org/fx"
)

// QuotesModule provides dependency injection for quote-related components.
var QuotesModule = fx.Module("quotes",
	// Include calculation service and filaments module
	calculationDI.Module,
	filamentDI.FilamentsModule,

	// Repositories
	fx.Provide(quotesRepositories.NewQuoteRepository),

	// Services
	fx.Provide(func(filamentRepo filamentsRepos.FilamentRepository) quotesServices.SnapshotService {
		return quotesServices.NewSnapshotService(filamentRepo)
	}),

	// Use Cases
	fx.Provide(quotesUseCases.NewQuoteUseCase),
)
