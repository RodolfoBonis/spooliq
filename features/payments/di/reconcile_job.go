package di

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	paymentUc "github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"github.com/getsentry/sentry-go"
	"go.uber.org/fx"
)

const (
	reconcileInterval     = 10 * time.Minute
	reconcileInitialDelay = time.Minute
	reconcileLookback     = 48 * time.Hour
	webhookMaxAttempts    = 10
	disconnectedRetention = 7 * 24 * time.Hour
	reconcileBatch        = 100
)

// RegisterReconcileJob wires the background payments reconciliation sweep: it
// reprocesses unprocessed webhook events, re-checks pending links (within validity +
// 48h grace), expires overdue links and purges credentials disconnected for over 7
// days. It holds an advisory lock so only one replica sweeps at a time, recovers from
// panics, and mirrors the budget expiry job's lifecycle.
func RegisterReconcileJob(lc fx.Lifecycle, repo domainRepos.PaymentRepository, confirm *paymentUc.ConfirmService, log logger.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(reconcileInterval)

	runSweep := func() {
		defer paymentUc.RecoverToSentry("payments.reconcile", log)
		acquired, err := repo.WithReconcileLock(ctx, func() error {
			now := time.Now()
			reprocessWebhooks(ctx, repo, confirm, log, now)
			recheckPendingLinks(ctx, repo, confirm, log)
			expireOverdueLinks(ctx, repo, log, now)
			purgeDisconnected(ctx, repo, log, now)
			return nil
		})
		if err != nil {
			log.Error(ctx, "reconcile sweep failed", map[string]interface{}{"error": err.Error()})
		}
		_ = acquired // when false, another replica holds the lock; nothing to do
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconcileInitialDelay):
					runSweep()
				}
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						runSweep()
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

func reprocessWebhooks(ctx context.Context, repo domainRepos.PaymentRepository, confirm *paymentUc.ConfirmService, log logger.Logger, now time.Time) {
	events, err := repo.ListUnprocessedWebhooks(ctx, now.Add(-reconcileLookback), webhookMaxAttempts, reconcileBatch)
	if err != nil {
		log.Error(ctx, "reconcile: list unprocessed webhooks failed", map[string]interface{}{"error": err.Error()})
		return
	}
	for _, ev := range events {
		var payload entities.WebhookPayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			_ = repo.MarkWebhookAttempt(ctx, ev.ID, "invalid payload")
			continue
		}
		if _, cerr := confirm.Confirm(ctx, ev.OrganizationID, payload, "", entities.SourceJob); cerr != nil {
			_ = repo.MarkWebhookAttempt(ctx, ev.ID, cerr.Error())
			if ev.Attempts+1 >= webhookMaxAttempts {
				alertExhausted(ev.OrganizationID, payload.OrderNSU, cerr)
				log.Error(ctx, "reconcile: webhook event exhausted attempts", map[string]interface{}{"event_id": ev.ID.String()})
			}
			continue
		}
		_ = repo.MarkWebhookProcessed(ctx, ev.ID)
	}
}

func recheckPendingLinks(ctx context.Context, repo domainRepos.PaymentRepository, confirm *paymentUc.ConfirmService, log logger.Logger) {
	pending, err := repo.ListPendingForRecheck(ctx, time.Now(), reconcileBatch)
	if err != nil {
		log.Error(ctx, "reconcile: list pending links failed", map[string]interface{}{"error": err.Error()})
		return
	}
	for _, p := range pending {
		payload := entities.WebhookPayload{OrderNSU: p.OrderNSU}
		if p.Slug != nil {
			payload.Slug = *p.Slug
		}
		if p.TransactionNSU != nil {
			payload.TransactionNSU = *p.TransactionNSU
		}
		if _, cerr := confirm.Confirm(ctx, p.OrganizationID, payload, "", entities.SourceJob); cerr != nil {
			log.Error(ctx, "reconcile: pending link recheck failed", map[string]interface{}{"order_nsu": p.OrderNSU, "error": cerr.Error()})
		}
	}
}

func expireOverdueLinks(ctx context.Context, repo domainRepos.PaymentRepository, log logger.Logger, now time.Time) {
	expired, err := repo.ListExpiredPendingLinks(ctx, now, reconcileBatch)
	if err != nil {
		log.Error(ctx, "reconcile: list expired links failed", map[string]interface{}{"error": err.Error()})
		return
	}
	for _, p := range expired {
		// Guarded: only expire a row that is still pending (never overwrite paid).
		if _, err := repo.CancelPendingByID(ctx, p.ID, entities.ReasonExpired); err != nil {
			log.Error(ctx, "reconcile: expire link failed", map[string]interface{}{"payment_id": p.ID.String(), "error": err.Error()})
		}
	}
}

func purgeDisconnected(ctx context.Context, repo domainRepos.PaymentRepository, log logger.Logger, now time.Time) {
	stale, err := repo.ListDisconnectedBefore(ctx, now.Add(-disconnectedRetention))
	if err != nil {
		log.Error(ctx, "reconcile: list disconnected integrations failed", map[string]interface{}{"error": err.Error()})
		return
	}
	for _, integ := range stale {
		if err := repo.DeleteIntegration(ctx, integ.ID); err != nil {
			log.Error(ctx, "reconcile: purge disconnected integration failed", map[string]interface{}{"integration_id": integ.ID.String(), "error": err.Error()})
		}
	}
}

// alertExhausted reports to Sentry that a webhook event will no longer be retried.
func alertExhausted(organizationID, orderNSU string, cause error) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetTag("area", "payments.reconcile")
		scope.SetExtra("organization_id", organizationID)
		scope.SetExtra("order_nsu", orderNSU)
		if cause != nil {
			scope.SetExtra("cause", cause.Error())
		}
		sentry.CaptureMessage("payments: webhook event exhausted retries")
	})
}
