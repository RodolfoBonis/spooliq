// Package di wires the filament stock feature for FX dependency injection.
package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/stock/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/stock/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the filament stock dependencies.
//
// The StockDeductor provider returns the stock repository as the budget feature's
// StockDeductor port, so budget completion can deduct filament stock in the same
// transaction without the budget package importing this feature.
var Module = fx.Module("stock", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.StockRepository {
		return repositories.NewStockRepository(db)
	}),
	fx.Annotate(func(repo domainRepositories.StockRepository, logger logger.Logger, activityService activityUc.IActivityService) usecases.IStockUseCase {
		return usecases.NewStockUseCase(repo, logger, activityService)
	}),
	func(repo domainRepositories.StockRepository) budgetUc.StockDeductor {
		return repo
	},
))
