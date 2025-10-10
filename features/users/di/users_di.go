package di

import (
	"github.com/RodolfoBonis/spooliq/features/users/data/repositories"
	domainRepositories "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// UsersModule provides the fx module for users dependencies.
var UsersModule = fx.Module("users",
	fx.Provide(
		func(db *gorm.DB) domainRepositories.UserRepository {
			return repositories.NewUserRepository(db)
		},
	),
)
