package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
)

// Stable, snake_case error codes surfaced by the slicer feature. They are the
// machine-readable contract the web app branches on.
const (
	// CodeFileRequired (400) — the multipart upload is missing the file part.
	CodeFileRequired = "file_required"
	// CodeUnsupportedFileFormat (400) — not a .gcode, .3mf or .gcode.3mf.
	CodeUnsupportedFileFormat = "unsupported_file_format"
	// CodeFileTooLarge (413) — the file exceeds the 200MB limit.
	CodeFileTooLarge = "file_too_large"
	// CodeFileNotSliced (422) — the file carries no slicing data.
	CodeFileNotSliced = "file_not_sliced"
	// CodeInvalidFile (400) — a corrupt/unreadable archive.
	CodeInvalidFile = "invalid_file"
	// CodeSliceAnalysisFailed (422) — analysis panicked or timed out.
	CodeSliceAnalysisFailed = "slice_analysis_failed"
	// CodeOrganizationRequired (400) — the organization ID is missing from context.
	CodeOrganizationRequired = "organization_required"
)

func errFileRequired() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeFileRequired, "O arquivo é obrigatório")
}

func errUnsupportedFileFormat() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeUnsupportedFileFormat, "Formato de arquivo não suportado. Apenas .gcode, .3mf e .gcode.3mf são permitidos")
}

func errFileTooLarge() *coreErrors.APIError {
	return &coreErrors.APIError{Status: http.StatusRequestEntityTooLarge, Code: CodeFileTooLarge, Message: "O arquivo excede o limite de 200MB"}
}

func errFileNotSliced() *coreErrors.APIError {
	return &coreErrors.APIError{Status: http.StatusUnprocessableEntity, Code: CodeFileNotSliced, Message: "O arquivo 3MF não contém dados de fatiamento. Fatie no seu slicer e envie o .gcode ou .gcode.3mf."}
}

func errInvalidFile() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeInvalidFile, "Arquivo inválido ou corrompido")
}

func errSliceAnalysisFailed() *coreErrors.APIError {
	return &coreErrors.APIError{Status: http.StatusUnprocessableEntity, Code: CodeSliceAnalysisFailed, Message: "Não foi possível analisar o arquivo de fatiamento"}
}

func errOrganizationRequired() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeOrganizationRequired, "Organização não encontrada no contexto")
}
