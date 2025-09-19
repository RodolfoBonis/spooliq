package export

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/export/domain/services"
	"github.com/RodolfoBonis/spooliq/features/export/presentation/handlers"
	"github.com/gin-gonic/gin"
)

// ExportQuotePDFHandler handles exporting quote as PDF.
// @Summary Export quote as PDF
// @Schemes
// @Description Exports a quote and its calculation results as PDF
// @Tags Export
// @Accept json
// @Produce application/pdf
// @Param id path int true "Quote ID"
// @Param request body handlers.ExportRequestDTO true "Export options"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/pdf [post]
// @Security Bearer
func QuotePDFHandler(exportService services.ExportService) gin.HandlerFunc {
	handler := handlers.NewExportHandler(exportService, nil)
	return handler.ExportQuotePDF
}

// ExportQuoteCSVHandler handles exporting quote as CSV.
// @Summary Export quote as CSV
// @Schemes
// @Description Exports a quote and its calculation results as CSV
// @Tags Export
// @Accept json
// @Produce text/csv
// @Param id path int true "Quote ID"
// @Param request body handlers.ExportRequestDTO true "Export options"
// @Success 200 {file} binary "CSV file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/csv [post]
// @Security Bearer
func QuoteCSVHandler(exportService services.ExportService) gin.HandlerFunc {
	handler := handlers.NewExportHandler(exportService, nil)
	return handler.ExportQuoteCSV
}

// ExportQuoteJSONHandler handles exporting quote as JSON.
// @Summary Export quote as JSON
// @Schemes
// @Description Exports a quote and its calculation results as JSON
// @Tags Export
// @Accept json
// @Produce application/json
// @Param id path int true "Quote ID"
// @Param request body handlers.ExportRequestDTO true "Export options"
// @Success 200 {file} binary "JSON file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/json [post]
// @Security Bearer
func QuoteJSONHandler(exportService services.ExportService) gin.HandlerFunc {
	handler := handlers.NewExportHandler(exportService, nil)
	return handler.ExportQuoteJSON
}

// GetSupportedFormatsHandler handles getting supported export formats.
// @Summary Get supported export formats
// @Schemes
// @Description Returns the list of supported export formats
// @Tags Export
// @Produce json
// @Success 200 {object} handlers.SupportedFormatsResponse "Successfully retrieved supported formats"
// @Router /exports/formats [get]
func GetSupportedFormatsHandler(exportService services.ExportService) gin.HandlerFunc {
	handler := handlers.NewExportHandler(exportService, nil)
	return handler.GetSupportedFormats
}

// Routes registers export routes for the application.
func Routes(route *gin.RouterGroup, exportService services.ExportService, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	// Export-specific routes
	exports := route.Group("/exports")
	exports.GET("/formats", GetSupportedFormatsHandler(exportService))

	// Quote export routes (these should be registered within quotes group in the main router)
	quotes := route.Group("/quotes")
	quotes.POST("/:id/export/pdf", protectFactory(QuotePDFHandler(exportService), roles.UserRole))
	quotes.POST("/:id/export/csv", protectFactory(QuoteCSVHandler(exportService), roles.UserRole))
	quotes.POST("/:id/export/json", protectFactory(QuoteJSONHandler(exportService), roles.UserRole))
}
