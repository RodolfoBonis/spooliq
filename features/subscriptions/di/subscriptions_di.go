package di

import (
	"github.com/RodolfoBonis/spooliq/features/subscriptions/data/repositories"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the fx module for subscriptions dependencies
var Module = fx.Module("subscriptions",
	fx.Provide(
		// Repositories
		func(db *gorm.DB) *repositories.SubscriptionRepositoryImpl {
			return repositories.NewSubscriptionRepository(db).(*repositories.SubscriptionRepositoryImpl)
		},
	),
)
