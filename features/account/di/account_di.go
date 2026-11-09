package di

import (
	"context"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/account"
	"github.com/RodolfoBonis/spooliq/features/account/domain/usecases"
	userRepos "github.com/RodolfoBonis/spooliq/features/users/domain/repositories"
	"go.uber.org/fx"
)

// AccountModule provides the self-service account endpoints.
var AccountModule = fx.Module("account",
	fx.Provide(
		func(
			keycloak services.IKeycloakAdminService,
			users userRepos.UserRepository,
			authService *services.AuthService,
			redis *services.RedisService,
			log logger.Logger,
		) *usecases.AccountUseCase {
			return usecases.NewAccountUseCase(
				keycloak,
				users,
				passwordVerifier(authService),
				redisRateLimiter(redis),
				log,
			)
		},
		account.NewHandler,
	),
)

// passwordVerifier checks credentials with a regular Keycloak login.
func passwordVerifier(authService *services.AuthService) usecases.PasswordVerifier {
	kc := config.EnvKeyCloak()
	return func(ctx context.Context, email, password string) error {
		_, err := authService.GetClient().Login(ctx, kc.ClientID, kc.ClientSecret, kc.Realm, email, password)
		return err
	}
}

// redisRateLimiter is a fixed-window limiter (INCR + EXPIRE). It fails open:
// a Redis outage never blocks the endpoint.
func redisRateLimiter(redis *services.RedisService) usecases.RateLimiter {
	return func(ctx context.Context, key string, limit int, window time.Duration) bool {
		if redis == nil || redis.GetClient() == nil {
			return true
		}
		client := redis.GetClient()
		count, err := client.Incr(ctx, "rl:"+key).Result()
		if err != nil {
			return true
		}
		if count == 1 {
			_ = client.Expire(ctx, "rl:"+key, window).Err()
		}
		return count <= int64(limit)
	}
}
