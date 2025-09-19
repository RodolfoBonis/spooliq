package di

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/RodolfoBonis/spooliq/features/export/domain/services"
	"github.com/RodolfoBonis/spooliq/features/export/presentation/handlers"
)

// Module provides the export feature module
var Module = fx.Module("export",
	fx.Provide(
		services.NewExportService,
		handlers.NewExportHandler,
	),
	fx.Invoke(RegisterExportRoutes),
)

// RegisterExportRoutes registra as rotas de export
func RegisterExportRoutes(r *gin.Engine, handler *handlers.ExportHandler) {
	v1 := r.Group("/v1")
	{
		// Endpoints para export de quotes
		quotes := v1.Group("/quotes")
		{
			quotes.POST("/:id/export/pdf", handler.ExportQuotePDF)
			quotes.POST("/:id/export/csv", handler.ExportQuoteCSV)
			quotes.POST("/:id/export/json", handler.ExportQuoteJSON)
		}

		// Endpoints gerais de export
		exports := v1.Group("/exports")
		{
			exports.GET("/formats", handler.GetSupportedFormats)
		}
	}
}
