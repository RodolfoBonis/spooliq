package entities

import (
	"time"

	quoteEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	calculationEntities "github.com/RodolfoBonis/spooliq/features/calculation/domain/entities"
)

// ExportFormat define os formatos de export suportados
type ExportFormat string

const (
	ExportFormatPDF  ExportFormat = "pdf"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatJSON ExportFormat = "json"
)

// ExportData contém todos os dados necessários para gerar um export
type ExportData struct {
	Quote       *quoteEntities.Quote                       `json:"quote"`
	Calculation *calculationEntities.CalculationResults    `json:"calculation,omitempty"`
	Metadata    *ExportMetadata                            `json:"metadata"`
}

// ExportMetadata contém metadados sobre o export
type ExportMetadata struct {
	GeneratedAt   time.Time    `json:"generated_at"`
	GeneratedBy   string       `json:"generated_by"`
	Format        ExportFormat `json:"format"`
	Version       string       `json:"version"`
	SystemInfo    SystemInfo   `json:"system_info"`
}

// SystemInfo contém informações do sistema
type SystemInfo struct {
	AppName    string `json:"app_name"`
	AppVersion string `json:"app_version"`
	Generator  string `json:"generator"`
}

// ExportRequest representa uma solicitação de export
type ExportRequest struct {
	QuoteID           uint         `json:"quote_id" validate:"required"`
	Format            ExportFormat `json:"format" validate:"required,oneof=pdf csv json"`
	IncludeCalculation bool         `json:"include_calculation"`
	CustomTitle       string       `json:"custom_title,omitempty"`
	Notes             string       `json:"notes,omitempty"`
}

// ExportResult representa o resultado de um export
type ExportResult struct {
	Data        []byte       `json:"data"`
	ContentType string       `json:"content_type"`
	Filename    string       `json:"filename"`
	Size        int64        `json:"size"`
	Format      ExportFormat `json:"format"`
}

// IsValidFormat verifica se o formato é válido
func (f ExportFormat) IsValid() bool {
	switch f {
	case ExportFormatPDF, ExportFormatCSV, ExportFormatJSON:
		return true
	default:
		return false
	}
}

// GetContentType retorna o Content-Type HTTP para o formato
func (f ExportFormat) GetContentType() string {
	switch f {
	case ExportFormatPDF:
		return "application/pdf"
	case ExportFormatCSV:
		return "text/csv"
	case ExportFormatJSON:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// GetFileExtension retorna a extensão do arquivo para o formato
func (f ExportFormat) GetFileExtension() string {
	switch f {
	case ExportFormatPDF:
		return "pdf"
	case ExportFormatCSV:
		return "csv"
	case ExportFormatJSON:
		return "json"
	default:
		return "bin"
	}
}

// Validate valida os dados de export
func (ed *ExportData) Validate() error {
	if ed.Quote == nil {
		return ErrInvalidExportData
	}

	if ed.Metadata == nil {
		return ErrMissingMetadata
	}

	if !ed.Metadata.Format.IsValid() {
		return ErrInvalidFormat
	}

	return nil
}

// GenerateFilename gera um nome de arquivo baseado na quote e formato
func (ed *ExportData) GenerateFilename() string {
	timestamp := ed.Metadata.GeneratedAt.Format("20060102_150405")
	quoteTitle := "orcamento"

	if ed.Quote.Title != "" {
		quoteTitle = ed.Quote.Title
	}

	return sanitizeFilename(quoteTitle + "_" + timestamp + "." + ed.Metadata.Format.GetFileExtension())
}

// sanitizeFilename remove caracteres inválidos do nome do arquivo
func sanitizeFilename(filename string) string {
	// Implementação simples - em produção usar biblioteca mais robusta
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range invalidChars {
		result = replaceAll(result, char, "_")
	}

	return result
}

// replaceAll substitui todas as ocorrências de old por new em s
func replaceAll(s, old, new string) string {
	result := ""
	for _, char := range s {
		if string(char) == old {
			result += new
		} else {
			result += string(char)
		}
	}
	return result
}

// Erros específicos do domínio de export
var (
	ErrInvalidExportData = &ExportError{Code: "INVALID_EXPORT_DATA", Message: "Dados de export inválidos"}
	ErrMissingMetadata   = &ExportError{Code: "MISSING_METADATA", Message: "Metadados obrigatórios ausentes"}
	ErrInvalidFormat     = &ExportError{Code: "INVALID_FORMAT", Message: "Formato de export inválido"}
	ErrExportGeneration  = &ExportError{Code: "EXPORT_GENERATION", Message: "Erro ao gerar export"}
)

// ExportError representa um erro específico do domínio de export
type ExportError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ExportError) Error() string {
	return e.Message
}