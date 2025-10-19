package di

import (
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
		usecases.NewPaymentMethodUseCase,
		usecases.NewSubscriptionPlanUseCase,
		usecases.NewManageSubscriptionUseCase,
	),
)
