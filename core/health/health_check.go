package health

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Routes registers the health check routes for the application.
func Routes(route *gin.RouterGroup, logger logger.Logger) {
	route.GET("/health_check", func(context *gin.Context) {
		// Create a manual trace to test OpenTelemetry integration
		tracer := otel.Tracer("spooliq-api")
		ctx, span := tracer.Start(context.Request.Context(), "health_check_manual")
		defer span.End()

		// Add attributes to the span
		span.SetAttributes(
			attribute.String("service.name", "spooliq-api"),
			attribute.String("endpoint", "/v1/health_check"),
			attribute.String("method", "GET"),
			attribute.Bool("manual_span", true),
		)

		logger.Info(ctx, "Health check accessed - manual trace created")
		context.String(http.StatusOK, "This Service is Healthy")
	})
}
