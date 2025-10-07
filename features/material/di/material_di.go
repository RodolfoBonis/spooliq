package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/material/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/material/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/material/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all material-related dependencies for FX dependency injection.
var Module = fx.Module("material", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.MaterialRepository { return repositories.NewMaterialRepository(db) }),
	fx.Annotate(func(repository domainRepositories.MaterialRepository, logger logger.Logger) usecases.IMaterialUseCase {
		return usecases.NewMaterialUseCase(repository, logger)
	}),
))
