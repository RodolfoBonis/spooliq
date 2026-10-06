package usecases

import (
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/gin-gonic/gin"
)

// Stable, snake_case error codes surfaced by the model3d feature. They are the
// machine-readable contract the web app branches on. The HTTP status of each is
// fixed by the constructor used below.
const (
	// CodeInvalidModel3DID (400) — the path model ID is not a valid UUID.
	CodeInvalidModel3DID = "invalid_model3d_id"
	// CodeModel3DNotFound (404) — the model does not exist in the organization.
	CodeModel3DNotFound = "model3d_not_found"
	// CodeFileRequired (400) — the multipart upload is missing the file part.
	CodeFileRequired = "file_required"
	// CodeUnsupportedFileFormat (400) — the file is not a .stl or .3mf.
	CodeUnsupportedFileFormat = "unsupported_file_format"
	// CodeFileTooLarge (400) — the file exceeds the 50MB limit.
	CodeFileTooLarge = "file_too_large"
	// CodeCustomerNotFound (404) — the referenced customer is not in the organization.
	CodeCustomerNotFound = "customer_not_found"
	// CodeModel3DDuplicate (409) — a model with the same file content already exists.
	CodeModel3DDuplicate = "model3d_duplicate"
	// CodeOrganizationRequired (400) — the organization ID is missing from context.
	CodeOrganizationRequired = "organization_required"
	// CodeInvalidCustomerID (400) — the customer_id is not a valid UUID.
	CodeInvalidCustomerID = "invalid_customer_id"
	// CodeInvalidFormat (400) — the format filter is not a known file format.
	CodeInvalidFormat = "invalid_format"
)

func errInvalidModel3DID() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeInvalidModel3DID, "ID de modelo 3D inválido")
}

func errModel3DNotFound() *coreErrors.APIError {
	return coreErrors.NotFoundErr(CodeModel3DNotFound, "Modelo 3D não encontrado")
}

func errFileRequired() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeFileRequired, "O arquivo é obrigatório")
}

func errUnsupportedFileFormat() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeUnsupportedFileFormat, "Formato de arquivo não suportado. Apenas .stl e .3mf são permitidos")
}

func errFileTooLarge() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeFileTooLarge, "O arquivo excede o limite de 50MB")
}

func errCustomerNotFound() *coreErrors.APIError {
	return coreErrors.NotFoundErr(CodeCustomerNotFound, "Cliente não encontrado")
}

func errOrganizationRequired() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeOrganizationRequired, "Organização não encontrada no contexto")
}

func errInvalidCustomerID() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeInvalidCustomerID, "ID de cliente inválido")
}

func errInvalidFormat() *coreErrors.APIError {
	return coreErrors.BadRequest(CodeInvalidFormat, "Formato inválido. Use .stl ou .3mf")
}

// respondDuplicate writes the 409 duplicate envelope. It keeps the standard
// {error, message, code} shape AND adds the existing model so the web app can link
// to it (it reads data.existing).
func respondDuplicate(c *gin.Context, existing *entities.Model3DEntity) {
	const msg = "Já existe um modelo com o mesmo conteúdo de arquivo"
	c.JSON(409, entities.DuplicateModel3DResponse{
		Error:    msg,
		Message:  msg,
		Code:     CodeModel3DDuplicate,
		Existing: existing,
	})
}
