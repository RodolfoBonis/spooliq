package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/usecases"
	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/gorm"
	"go.uber.org/fx"
)

// Module provides dependency injection for filament metadata feature
var Module = fx.Module("filament-metadata",
	fx.Provide(
		// Repositories
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.BrandRepository {
				return repositories.NewBrandRepository(db)
			},
		),
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.MaterialRepository {
				return repositories.NewMaterialRepository(db)
			},
		),

		// Use Cases
		fx.Annotate(
			func(brandRepo domainRepositories.BrandRepository, logger logger.Logger) usecases.BrandUseCase {
				return usecases.NewBrandUseCase(brandRepo, validator.New(), logger)
			},
		),
		fx.Annotate(
			func(materialRepo domainRepositories.MaterialRepository, logger logger.Logger) usecases.MaterialUseCase {
				return usecases.NewMaterialUseCase(materialRepo, validator.New(), logger)
			},
		),
	),
)
