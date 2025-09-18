package di

import (
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	"go.uber.org/fx"
)

// AuthModule provides the fx module for authentication dependencies.
var AuthModule = fx.Module("auth",
	fx.Provide(
		func(authService *services.AuthService, logger logger.Logger) usecases.AuthUseCase {
			return usecases.NewAuthUseCase(authService.GetClient(), config.EnvKeyCloak(), logger)
		},
	),
)
