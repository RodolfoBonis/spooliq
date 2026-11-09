package di

import (
	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/crypto"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	coreServices "github.com/RodolfoBonis/spooliq/core/services"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	budgetUc "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/payments"
	dataRepos "github.com/RodolfoBonis/spooliq/features/payments/data/repositories"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module exports the payments feature's dependency injection module.
var Module = fx.Module(
	"payments",
	fx.Provide(
		NewSecretBox,
		func(db *gorm.DB) domainRepos.PaymentRepository {
			return dataRepos.NewPaymentRepository(db)
		},
		// Shared service: wires the dashboard cache invalidator from the cache
		// middleware (nil-safe).
		func(
			repo domainRepos.PaymentRepository,
			client coreServices.IInfinitePayClient,
			box *crypto.SecretBox,
			cfg *config.AppConfig,
			notifications notificationUc.INotificationService,
			activity activityUc.IActivityService,
			cache *middlewares.CacheMiddleware,
			log logger.Logger,
		) *usecases.Service {
			var invalidator budgetUc.DashboardCacheInvalidator
			if cache != nil {
				invalidator = cache
			}
			return usecases.NewService(repo, client, box, cfg, notifications, activity, invalidator, log)
		},
		// Adapter: expose the budget public UC as the narrow PublicViewBuilder so the
		// public confirm endpoint can return the full budget view.
		func(pub budgetUc.IPublicBudgetUseCase) budgetUc.PublicViewBuilder {
			if concrete, ok := pub.(*budgetUc.PublicBudgetUseCase); ok {
				return concrete
			}
			return nil
		},
		usecases.NewChargeService,
		usecases.NewConfirmService,
		usecases.NewIntegrationUseCase,
		usecases.NewPublicPaymentProvider,
		usecases.NewBudgetPaymentsUseCase,
		usecases.NewPublicPaymentUseCase,
		usecases.NewWebhookUseCase,
		payments.NewHandler,
		// Budget-facing ports (implemented here so budget never imports payments).
		NewDefaultsProvider,
		NewPendingCanceler,
	),
	// Wire the public payment block into the budget public view, and start the
	// reconcile job.
	fx.Invoke(func(pub budgetUc.IPublicBudgetUseCase, provider *usecases.PublicPaymentProvider) {
		if concrete, ok := pub.(*budgetUc.PublicBudgetUseCase); ok {
			concrete.SetPaymentProvider(provider)
		}
	}),
	fx.Invoke(RegisterReconcileJob),
)
