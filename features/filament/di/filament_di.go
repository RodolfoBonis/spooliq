package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/filament/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/filament/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all filament-related dependencies for FX dependency injection.
var Module = fx.Module("filament", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.FilamentRepository { return repositories.NewFilamentRepository(db) }),
	fx.Annotate(func(repository domainRepositories.FilamentRepository, logger logger.Logger) usecases.IFilamentUseCase {
		return usecases.NewFilamentUseCase(repository, logger)
	}),
))
