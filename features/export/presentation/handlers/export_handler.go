package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/export/domain/services"
	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/go-playground/validator/v10"
)

type ExportHandler struct {
	exportService services.ExportService
	logger        logger.Logger
	validator     *validator.Validate
}

// NewExportHandler cria uma nova instância do handler de export
func NewExportHandler(
	exportService services.ExportService,
	logger logger.Logger,
) *ExportHandler {
	return &ExportHandler{
		exportService: exportService,
		logger:        logger,
		validator:     validator.New(),
	}
}

// ExportQuotePDF exporta uma quote em formato PDF
// @Summary Export quote as PDF
// @Description Exports a quote and its calculation results as PDF
// @Tags exports
// @Accept json
// @Produce application/pdf
// @Param id path int true "Quote ID"
// @Param request body ExportRequestDTO true "Export options"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/pdf [post]
func (h *ExportHandler) ExportQuotePDF(c *gin.Context) {
	h.exportQuote(c, entities.ExportFormatPDF)
}

// ExportQuoteCSV exporta uma quote em formato CSV
// @Summary Export quote as CSV
// @Description Exports a quote and its calculation results as CSV
// @Tags exports
// @Accept json
// @Produce text/csv
// @Param id path int true "Quote ID"
// @Param request body ExportRequestDTO true "Export options"
// @Success 200 {file} binary "CSV file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/csv [post]
func (h *ExportHandler) ExportQuoteCSV(c *gin.Context) {
	h.exportQuote(c, entities.ExportFormatCSV)
}

// ExportQuoteJSON exporta uma quote em formato JSON
// @Summary Export quote as JSON
// @Description Exports a quote and its calculation results as JSON
// @Tags exports
// @Accept json
// @Produce application/json
// @Param id path int true "Quote ID"
// @Param request body ExportRequestDTO true "Export options"
// @Success 200 {file} binary "JSON file"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /quotes/{id}/export/json [post]
func (h *ExportHandler) ExportQuoteJSON(c *gin.Context) {
	h.exportQuote(c, entities.ExportFormatJSON)
}

// GetSupportedFormats retorna os formatos de export suportados
// @Summary Get supported export formats
// @Description Returns the list of supported export formats
// @Tags exports
// @Produce json
// @Success 200 {object} SupportedFormatsResponse
// @Router /exports/formats [get]
func (h *ExportHandler) GetSupportedFormats(c *gin.Context) {
	formats := h.exportService.GetSupportedFormats()

	response := SupportedFormatsResponse{
		Formats: make([]FormatInfo, len(formats)),
	}

	for i, format := range formats {
		response.Formats[i] = FormatInfo{
			Format:      string(format),
			ContentType: format.GetContentType(),
			Extension:   format.GetFileExtension(),
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *ExportHandler) exportQuote(c *gin.Context, format entities.ExportFormat) {
	// Obter ID da quote
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "ID inválido", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Invalid quote ID for export", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Obter dados da request
	var req ExportRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Formato de requisição inválido", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to bind export request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Validar request
	if err := h.validator.Struct(&req); err != nil {
		appError := errors.NewAppError(coreEntities.ErrEntity, "Falha na validação dos dados", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Validation failed for export request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Obter user ID
	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(coreEntities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "User not authenticated for export", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Criar request de export
	exportRequest := &entities.ExportRequest{
		QuoteID:           uint(id),
		Format:            format,
		IncludeCalculation: req.IncludeCalculation,
		CustomTitle:       req.CustomTitle,
		Notes:             req.Notes,
	}

	// Executar export
	result, err := h.exportService.ExportQuote(c.Request.Context(), exportRequest, userID)
	if err != nil {
		appError := errors.NewAppError(coreEntities.ErrService, "Erro ao gerar export", nil, err)
		httpError := appError.ToHTTPError()
		h.logger.LogError(c.Request.Context(), "Failed to export quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Configurar headers de download
	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", "attachment; filename=\""+result.Filename+"\"")
	c.Header("Content-Length", strconv.FormatInt(result.Size, 10))

	// Enviar arquivo
	c.Data(http.StatusOK, result.ContentType, result.Data)

	h.logger.Info(c.Request.Context(), "Quote exported successfully", map[string]interface{}{
		"quote_id": id,
		"format":   string(format),
		"size":     result.Size,
		"filename": result.Filename,
		"user_id":  userID,
	})
}

// DTOs para requests e responses

type ExportRequestDTO struct {
	IncludeCalculation bool   `json:"include_calculation"`
	CustomTitle       string `json:"custom_title,omitempty"`
	Notes             string `json:"notes,omitempty"`
}

type SupportedFormatsResponse struct {
	Formats []FormatInfo `json:"formats"`
}

type FormatInfo struct {
	Format      string `json:"format"`
	ContentType string `json:"content_type"`
	Extension   string `json:"extension"`
}