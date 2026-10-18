package di

import (
	"context"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/middlewares"
	budgetRepos "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	budgetUsecases "github.com/RodolfoBonis/spooliq/features/budget/domain/usecases"
	notificationUc "github.com/RodolfoBonis/spooliq/features/notification/domain/usecases"
	"go.uber.org/fx"
)

// expiryJobInterval is how often the budget expiry sweep runs.
const expiryJobInterval = time.Hour

// expiryJobInitialDelay postpones the first sweep. Migrations run in an OnStart hook
// registered after this module's, so sweeping immediately would hit the schema before
// new columns (e.g. valid_until) exist on the first deploy of a release.
const expiryJobInitialDelay = time.Minute

// RegisterExpiryJob wires a background goroutine that periodically marks overdue
// "sent" budgets as "expired". It runs once shortly after start and then every hour,
// and is stopped cleanly via the OnStop context so it never leaks past shutdown.
//
// The public endpoints also check valid_until directly, so correctness never
// depends on this job having run — it only keeps the stored status in sync for the
// dashboard/list views.
func RegisterExpiryJob(lc fx.Lifecycle, repo budgetRepos.BudgetRepository, notifications notificationUc.INotificationService, cache *middlewares.CacheMiddleware, log logger.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(expiryJobInterval)

	runOnce := func() {
		expired, err := repo.ExpireOverdue(ctx)
		if err != nil {
			log.Error(ctx, "Budget expiry sweep failed", map[string]interface{}{"error": err.Error()})
			return
		}
		if len(expired) > 0 {
			log.Info(ctx, "Budget expiry sweep completed", map[string]interface{}{"expired_count": len(expired)})
		}
		orgs := map[string]bool{}
		for _, b := range expired {
			notifications.Notify(b.OrganizationID, budgetUsecases.ExpiredNotification(b))
			orgs[b.OrganizationID] = true
		}
		for orgID := range orgs {
			if cache != nil {
				budgetUsecases.InvalidateDashboard(ctx, cache, log, orgID)
			}
		}
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(expiryJobInitialDelay):
					runOnce()
				}
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						runOnce()
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			ticker.Stop()
			return nil
		},
	})
}
