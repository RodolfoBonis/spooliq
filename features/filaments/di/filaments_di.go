package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	data_repositories "github.com/RodolfoBonis/spooliq/features/filaments/data/repositories"
	domain_repositories "github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/usecases"
	metadataRepos "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/repositories"
	"github.com/jinzhu/gorm"
	"go.uber.org/fx"
)

// FilamentsModule provides dependency injection for filament-related components.
var FilamentsModule = fx.Module("filaments",
	fx.Provide(
		func(db *gorm.DB) domain_repositories.FilamentRepository {
			return data_repositories.NewFilamentRepository(db)
		},
		func(filamentRepo domain_repositories.FilamentRepository, brandRepo metadataRepos.BrandRepository, materialRepo metadataRepos.MaterialRepository, logger logger.Logger) usecases.FilamentUseCase {
			return usecases.NewFilamentUseCase(filamentRepo, brandRepo, materialRepo, logger)
		},
	),
)
