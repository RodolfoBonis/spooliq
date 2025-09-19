package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	data_repositories "github.com/RodolfoBonis/spooliq/features/filaments/data/repositories"
	domain_repositories "github.com/RodolfoBonis/spooliq/features/filaments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/usecases"
	"go.uber.org/fx"
	"github.com/jinzhu/gorm"
)

var FilamentsModule = fx.Module("filaments",
	fx.Provide(
		func(db *gorm.DB) domain_repositories.FilamentRepository {
			return data_repositories.NewFilamentRepository(db)
		},
		func(filamentRepo domain_repositories.FilamentRepository, logger logger.Logger) usecases.FilamentUseCase {
			return usecases.NewFilamentUseCase(filamentRepo, logger)
		},
	),
)