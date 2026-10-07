package di

import (
	"context"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	budgetRepos "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"go.uber.org/fx"
)

// expiryJobInterval is how often the budget expiry sweep runs.
const expiryJobInterval = time.Hour

// RegisterExpiryJob wires a background goroutine that periodically marks overdue
// "sent" budgets as "expired". It runs once right after start and then every hour,
// and is stopped cleanly via the OnStop context so it never leaks past shutdown.
//
// The public endpoints also check valid_until directly, so correctness never
// depends on this job having run — it only keeps the stored status in sync for the
// dashboard/list views.
func RegisterExpiryJob(lc fx.Lifecycle, repo budgetRepos.BudgetRepository, log logger.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(expiryJobInterval)

	runOnce := func() {
		count, err := repo.ExpireOverdue(ctx)
		if err != nil {
			log.Error(ctx, "Budget expiry sweep failed", map[string]interface{}{"error": err.Error()})
			return
		}
		if count > 0 {
			log.Info(ctx, "Budget expiry sweep completed", map[string]interface{}{"expired_count": count})
		}
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				runOnce() // one run on start
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
