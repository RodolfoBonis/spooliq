package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/supply/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/supply/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/supply/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all supply-related dependencies for FX dependency injection.
var Module = fx.Module("supply", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.SupplyRepository { return repositories.NewSupplyRepository(db) }),
	fx.Annotate(func(repository domainRepositories.SupplyRepository, logger logger.Logger, activityService activityUc.IActivityService) usecases.ISupplyUseCase {
		return usecases.NewSupplyUseCase(repository, logger, activityService)
	}),
))
