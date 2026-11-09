package di

import (
	"context"
	"encoding/json"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	paymentUc "github.com/RodolfoBonis/spooliq/features/payments/domain/usecases"
	"go.uber.org/fx"
)

const (
	// reconcileInterval is how often the payments reconcile sweep runs.
	reconcileInterval = 10 * time.Minute
	// reconcileInitialDelay postpones the first sweep so migrations have run.
	reconcileInitialDelay = time.Minute
	// reconcileLookback bounds how far back the job reprocesses events/links.
	reconcileLookback = 48 * time.Hour
	// webhookMaxAttempts caps webhook event reprocessing.
	webhookMaxAttempts = 10
	// disconnectedRetention is how long disconnected credentials are kept before purge.
	disconnectedRetention = 7 * 24 * time.Hour
	// reconcileBatch bounds each sweep's work.
	reconcileBatch = 100
)

// RegisterReconcileJob wires the background payments reconciliation sweep: it
// reprocesses unprocessed webhook events, re-checks pending links, expires overdue
// links and purges credentials disconnected for over 7 days. It mirrors the budget
// expiry job's lifecycle (one run shortly after start, then every 10 minutes, cleanly
// stopped on shutdown).
func RegisterReconcileJob(lc fx.Lifecycle, repo domainRepos.PaymentRepository, confirm *paymentUc.ConfirmService, log logger.Logger) {
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(reconcileInterval)

	runOnce := func() {
		now := time.Now()
		reprocessWebhooks(ctx, repo, confirm, log, now)
		recheckPendingLinks(ctx, repo, confirm, log, now)
		expireOverdueLinks(ctx, repo, log, now)
		purgeDisconnected(ctx, repo, log, now)
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(reconcileInitialDelay):
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

// reprocessWebhooks re-confirms unprocessed webhook events.
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
				log.Error(ctx, "reconcile: webhook event exhausted attempts", map[string]interface{}{"event_id": ev.ID.String()})
			}
			continue
		}
		_ = repo.MarkWebhookProcessed(ctx, ev.ID)
	}
}

// recheckPendingLinks re-confirms pending payments that carry a slug/transaction_nsu.
func recheckPendingLinks(ctx context.Context, repo domainRepos.PaymentRepository, confirm *paymentUc.ConfirmService, log logger.Logger, now time.Time) {
	pending, err := repo.ListPendingWithIdentifiers(ctx, now.Add(-reconcileLookback), reconcileBatch)
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
			log.Error(ctx, "reconcile: pending link recheck failed", map[string]interface{}{"payment_order_nsu": p.OrderNSU, "error": cerr.Error()})
		}
	}
}

// expireOverdueLinks cancels pending links past their validity.
func expireOverdueLinks(ctx context.Context, repo domainRepos.PaymentRepository, log logger.Logger, now time.Time) {
	expired, err := repo.ListExpiredPendingLinks(ctx, now, reconcileBatch)
	if err != nil {
		log.Error(ctx, "reconcile: list expired links failed", map[string]interface{}{"error": err.Error()})
		return
	}
	for _, p := range expired {
		if err := repo.UpdatePaymentFields(ctx, p.ID, map[string]interface{}{
			"status":        string(entities.PaymentCanceled),
			"status_reason": entities.ReasonExpired,
			"canceled_at":   now,
		}); err != nil {
			log.Error(ctx, "reconcile: expire link failed", map[string]interface{}{"payment_id": p.ID.String(), "error": err.Error()})
		}
	}
}

// purgeDisconnected removes credentials disconnected for longer than the retention.
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
