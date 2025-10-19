package di

import (
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	companyRepositories "github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the fx module for subscriptions dependencies
var Module = fx.Module("subscriptions",
	fx.Provide(
		// Repositories
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.SubscriptionRepository {
				return repositories.NewSubscriptionRepository(db)
			},
			fx.As(new(domainRepositories.SubscriptionRepository)),
		),
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.PlanRepository {
				return repositories.NewPlanRepository(db)
			},
			fx.As(new(domainRepositories.PlanRepository)),
		),
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.PaymentMethodRepository {
				return repositories.NewPaymentMethodRepository(db)
			},
			fx.As(new(domainRepositories.PaymentMethodRepository)),
		),
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.SubscriptionPlanRepository {
				return repositories.NewSubscriptionPlanRepository(db)
			},
			fx.As(new(domainRepositories.SubscriptionPlanRepository)),
		),

		// Use Cases
		fx.Annotate(
			func(
				subscriptionRepo domainRepositories.SubscriptionRepository,
				planRepo domainRepositories.PlanRepository,
				companyRepo companyRepositories.CompanyRepository,
				asaasService services.IAsaasService,
				logger logger.Logger,
			) usecases.ISubscriptionUseCase {
				return usecases.NewSubscriptionUseCase(subscriptionRepo, planRepo, companyRepo, asaasService, logger)
			},
			fx.As(new(usecases.ISubscriptionUseCase)),
		),
		// PaymentMethodUseCase (concrete type, not interface)
		usecases.NewPaymentMethodUseCase,
		// SubscriptionPlanUseCase (concrete type, not interface)
		usecases.NewSubscriptionPlanUseCase,
		// ManageSubscriptionUseCase (concrete type, not interface)
		usecases.NewManageSubscriptionUseCase,
	),
)
