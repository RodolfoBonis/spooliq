package di

import (
	"go.uber.org/fx"

	"github.com/Nerzal/gocloak/v13"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/entities"
	userRepositories "github.com/RodolfoBonis/spooliq/features/users/data/repositories"
	userServices "github.com/RodolfoBonis/spooliq/features/users/data/services"
	"github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	domainServices "github.com/RodolfoBonis/spooliq/features/users/domain/services"
)

// Module provides the users feature module
var Module = fx.Module("users",
	fx.Provide(
		// Keycloak client
		func(keycloakConfig entities.KeyCloakDataEntity) *gocloak.GoCloak {
			return gocloak.NewClient(keycloakConfig.Host)
		},

		// Repositories
		fx.Annotate(
			userRepositories.NewKeycloakUserRepository,
			fx.As(new(repositories.UserRepository)),
		),

		// Services
		fx.Annotate(
			userServices.NewUserService,
			fx.As(new(domainServices.UserService)),
		),

		// Keycloak configuration provider
		func() entities.KeyCloakDataEntity {
			return config.EnvKeyCloak()
		},
	),
)
