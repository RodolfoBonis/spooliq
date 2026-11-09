package config

import (
	"fmt"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/getsentry/sentry-go"
)

// SentryConfig returns the Sentry configuration for the application.
func SentryConfig() {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              EnvSentryDSN(),
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		// Scrub every secret token (public share token, InfinitePay webhook token)
		// from errors AND performance transactions so it never leaves the process
		// embedded in a URL, transaction name or breadcrumb.
		BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			return scrubSensitivePaths(event)
		},
		BeforeSendTransaction: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
			return scrubSensitivePaths(event)
		},
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
		// Don't exit on Sentry failure, just log and continue
	}
}

// scrubSensitivePaths masks every secret path token everywhere it could appear in a
// Sentry event: the request URL, the transaction name and breadcrumb messages/urls.
// It is a no-op for events that never touched a masked route.
func scrubSensitivePaths(event *sentry.Event) *sentry.Event {
	if event == nil {
		return event
	}
	event.Transaction = helpers.MaskSensitivePath(event.Transaction)
	if event.Request != nil {
		event.Request.URL = helpers.MaskSensitivePath(event.Request.URL)
	}
	for _, crumb := range event.Breadcrumbs {
		if crumb == nil {
			continue
		}
		crumb.Message = helpers.MaskSensitivePath(crumb.Message)
		if crumb.Data != nil {
			if raw, ok := crumb.Data["url"].(string); ok {
				crumb.Data["url"] = helpers.MaskSensitivePath(raw)
			}
		}
	}
	return event
}
