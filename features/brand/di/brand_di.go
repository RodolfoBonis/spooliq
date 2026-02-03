package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/brand/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all brand-related dependencies for FX dependency injection.
var Module = fx.Module("brand", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.BrandRepository { return repositories.NewBrandRepository(db) }),
	fx.Annotate(func(repository domainRepositories.BrandRepository, logger logger.Logger, activityService activityUc.IActivityService) usecases.IBrandUseCase {
		return usecases.NewBrandUseCase(repository, logger, activityService)
	}),
))
