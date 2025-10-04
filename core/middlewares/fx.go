package middlewares

import (
	"go.uber.org/fx"
)

// Module provides the fx module for middlewares.
var Module = fx.Module("middlewares",
	fx.Provide(
		NewMonitoringMiddleware,
		NewProtectMiddleware,
		NewCacheMiddleware,
		// NewTracingMiddlewareProvider removed - now using observability.Instrumentor
	),
)
