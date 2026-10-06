// Package di wires the slicer feature for FX dependency injection.
package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/slicer/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/slicer/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all slicer-related dependencies. It also provides the
// service.Service, which the model3d feature consumes to analyze .3mf uploads.
var Module = fx.Module("slicer", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.FilamentCatalogRepository {
		return repositories.NewFilamentCatalogRepository(db)
	}),
	fx.Annotate(func(catalog domainRepositories.FilamentCatalogRepository, logger logger.Logger) service.Service {
		return service.NewService(catalog, logger)
	}),
	fx.Annotate(func(svc service.Service, logger logger.Logger) usecases.ISlicerUseCase {
		return usecases.NewSlicerUseCase(svc, logger)
	}),
))
