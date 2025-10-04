package health

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/gin-gonic/gin"
)

// Routes registers the health check routes for the application.
// Observability is now handled automatically by the HTTP instrumentor.
func Routes(route *gin.RouterGroup, logger logger.Logger) {
	route.GET("/health_check", func(context *gin.Context) {
		// Simple health check - automatic instrumentation handles tracing
		logger.Info(context.Request.Context(), "Health check accessed")
		context.String(http.StatusOK, "This Service is Healthy")
	})
}
