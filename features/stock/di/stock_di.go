// Package di wires the filament stock feature for FX dependency injection.
package di

import (
	"context"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/stock/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/stock/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/usecases"
	"github.com/google/uuid"
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
	fx.Annotate(func(repo domainRepositories.StockRepository, logger logger.Logger, activityService activityUc.IActivityService, notifications notificationUc.INotificationService) usecases.IStockUseCase {
		return usecases.NewStockUseCase(repo, logger, activityService, notifications)
	}),
	func(repo domainRepositories.StockRepository) budgetUc.StockDeductor {
		return stockDeductorAdapter{repo}
	},
))

// stockDeductorAdapter exposes the stock repository as the budget port,
// converting its types so the budget package never imports this feature.
type stockDeductorAdapter struct {
	domainRepositories.StockRepository
}

func (a stockDeductorAdapter) LowStockForBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]budgetUc.LowStockFilament, error) {
	rows, err := a.StockRepository.LowStockForBudget(ctx, budgetID, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]budgetUc.LowStockFilament, 0, len(rows))
	for _, r := range rows {
		out = append(out, budgetUc.LowStockFilament(r))
	}
	return out, nil
}
