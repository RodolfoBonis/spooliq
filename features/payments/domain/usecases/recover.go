package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/getsentry/sentry-go"
)

// RecoverToSentry recovers a panic in a detached goroutine (webhook confirm, reconcile
// sweep), logs it and reports it to Sentry so a background crash is never silent. Use
// it as `defer RecoverToSentry("payments.webhook", log)`.
func RecoverToSentry(area string, log logger.Logger) {
	rec := recover()
	if rec == nil {
		return
	}
	if log != nil {
		log.Error(context.Background(), "payments: recovered panic", map[string]interface{}{
			"area":  area,
			"panic": fmt.Sprintf("%v", rec),
		})
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("area", area)
		sentry.CurrentHub().Recover(rec)
	})
	sentry.Flush(2 * time.Second)
}
