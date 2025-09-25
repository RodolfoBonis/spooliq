package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/brand/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/brand/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/usecases"
	"github.com/jinzhu/gorm"
	"go.uber.org/fx"
)

var Module = fx.Module("brand", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.BrandRepository { return repositories.NewBrandRepository(db) }),
	fx.Annotate(func(repository domainRepositories.BrandRepository, logger logger.Logger) usecases.IBrandUseCase {
		return usecases.NewBrandUseCase(repository, logger)
	}),
))
