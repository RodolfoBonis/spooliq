package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/model3d/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/usecases"
	slicerservice "github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides all model3d-related dependencies for FX dependency injection.
var Module = fx.Module("model3d", fx.Provide(
	fx.Annotate(func(db *gorm.DB) domainRepositories.Model3DRepository {
		return repositories.NewModel3DRepository(db)
	}),
	fx.Annotate(func(repository domainRepositories.Model3DRepository, cdnService *services.CDNService, thumbnailService *services.ThumbnailService, slicerService slicerservice.Service, logger logger.Logger, activityService activityUc.IActivityService) usecases.IModel3DUseCase {
		return usecases.NewModel3DUseCase(repository, cdnService, thumbnailService, slicerService, logger, activityService)
	}),
))
