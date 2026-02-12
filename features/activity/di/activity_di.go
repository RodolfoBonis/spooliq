package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/activity/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/activity/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all activity-related dependencies for FX dependency injection.
var Module = fx.Module("activity", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.ActivityRepository {
		return repositories.NewActivityRepository(db)
	}),
	fx.Annotate(func(repository domainRepositories.ActivityRepository, logger logger.Logger) usecases.IActivityService {
		return usecases.NewActivityService(repository, logger)
	}),
))
