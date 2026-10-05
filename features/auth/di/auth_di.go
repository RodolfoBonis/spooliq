package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/auth/domain/usecases"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	presetUsecases "github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	userRepositories "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"go.uber.org/fx"
)

// AuthModule provides the fx module for authentication dependencies.
var AuthModule = fx.Module("auth",
	fx.Provide(
		func(authService *services.AuthService, logger logger.Logger) usecases.AuthUseCase {
			return usecases.NewAuthUseCase(authService.GetClient(), config.EnvKeyCloak(), logger)
		},
		func(
			keycloakService services.IKeycloakAdminService,
			asaasService services.IAsaasService,
			companyRepository companyRepositories.CompanyRepository,
			userRepository userRepositories.UserRepository,
			gatewayLinkRepo subscriptionRepositories.PaymentGatewayLinkRepository,
			presetCreateUC *presetUsecases.CreatePresetUseCase,
			logger logger.Logger,
		) *usecases.RegisterUseCase {
			return usecases.NewRegisterUseCase(keycloakService, asaasService, companyRepository, userRepository, gatewayLinkRepo, presetCreateUC, logger)
		},
	),
)
