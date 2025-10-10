package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/uploads/domain/usecases"
	"go.uber.org/fx"
)

// Module exports the uploads feature's dependencies
var Module = fx.Options(
	fx.Provide(func(cdnService *services.CDNService, logger logger.Logger) usecases.IUploadUseCase {
		return usecases.NewUploadUseCase(cdnService, logger)
	}),
)
