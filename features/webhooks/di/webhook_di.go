package di

import (
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/logger"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/webhooks"
	"github.com/RodolfoBonis/spooliq/features/webhooks/domain/usecases"
	"go.uber.org/fx"
)

// Module provides the fx module for webhooks
var Module = fx.Module("webhooks",
	fx.Provide(
		func(
			companyRepository companyRepositories.CompanyRepository,
			cfg *config.AppConfig,
			logger logger.Logger,
		) *usecases.AsaasWebhookUseCase {
			return usecases.NewAsaasWebhookUseCase(companyRepository, cfg, logger)
		},
		webhooks.NewWebhookHandler,
	),
)

