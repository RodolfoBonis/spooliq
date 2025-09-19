package di

import (
	calculationDI "github.com/RodolfoBonis/spooliq/features/calculation/di"
	quotesRepositories "github.com/RodolfoBonis/spooliq/features/quotes/data/repositories"
	quotesUseCases "github.com/RodolfoBonis/spooliq/features/quotes/domain/usecases"
	quotesHandlers "github.com/RodolfoBonis/spooliq/features/quotes/presentation/handlers"
	"github.com/gin-gonic/gin"
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

	// Handlers
	fx.Provide(quotesHandlers.NewQuoteHandler),

	// Register routes
	fx.Invoke(RegisterQuoteRoutes),
)

// RegisterQuoteRoutes registers all quote-related HTTP routes.
func RegisterQuoteRoutes(
	handler *quotesHandlers.QuoteHandler,
	router *gin.Engine,
) {
	quotesGroup := router.Group("/v1/quotes")
	{
		quotesGroup.POST("", handler.CreateQuote)
		quotesGroup.GET("", handler.GetUserQuotes)
		quotesGroup.GET("/:id", handler.GetQuote)
		quotesGroup.PUT("/:id", handler.UpdateQuote)
		quotesGroup.DELETE("/:id", handler.DeleteQuote)
		quotesGroup.POST("/:id/duplicate", handler.DuplicateQuote)
		quotesGroup.POST("/:id/calculate", handler.CalculateQuote)
	}
}
