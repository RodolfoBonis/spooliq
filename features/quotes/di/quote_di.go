package di

import (
	calculationDI "github.com/RodolfoBonis/spooliq/features/calculation/di"
	quotesRepositories "github.com/RodolfoBonis/spooliq/features/quotes/data/repositories"
	quotesUseCases "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	"go.uber.org/fx"
)

// QuotesModule provides dependency injection for quote-related components.
var QuotesModule = fx.Module("quotes",
	// Include calculation service
	calculationDI.Module,

	// Repositories
	fx.Provide(quotesRepositories.NewQuoteRepository),

	// Use Cases
	fx.Provide(quotesUseCases.NewQuoteUseCase),
)
