package di

import (
	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the fx module for subscriptions dependencies
var Module = fx.Module("subscriptions",
	fx.Provide(
		fx.Annotate(
			func(db *gorm.DB) domainRepositories.SubscriptionRepository {
				return repositories.NewSubscriptionRepository(db)
			},
			fx.As(new(domainRepositories.SubscriptionRepository)),
		),
	),
)
