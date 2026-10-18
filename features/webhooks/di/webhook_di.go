package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	subscriptionRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/webhooks"
	"github.com/RodolfoBonis/spooliq/features/webhooks/domain/usecases"
	"go.uber.org/fx"
)

// Module provides the fx module for webhooks
var Module = fx.Module("webhooks",
	fx.Provide(
		func(
			companyRepository companyRepositories.CompanyRepository,
			subscriptionRepository subscriptionRepositories.SubscriptionRepository,
			cfg *config.AppConfig,
			logger logger.Logger,
			notifications notificationUc.INotificationService,
		) *usecases.AsaasWebhookUseCase {
			return usecases.NewAsaasWebhookUseCase(companyRepository, subscriptionRepository, cfg, logger, notifications)
		},
		webhooks.NewWebhookHandler,
	),
)
