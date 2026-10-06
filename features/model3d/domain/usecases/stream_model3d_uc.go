package usecases

import (
	"errors"
	"mime"
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StreamFile streams the raw model file for a 3D model from the CDN (MinIO)
// through the API, organization-scoped. It sets a format-appropriate Content-Type
// (model/stl or model/3mf), the Content-Length, an inline Content-Disposition with
// the original file name, and a short private cache. A missing underlying object
// yields a 404.
// @Summary Stream 3D Model file
// @Description Stream the raw STL/3MF file for a 3D model.
// @Tags models3d
// @Produce application/octet-stream
// @Param id path string true "3D Model ID" format(uuid)
// @Success 200 {file} file "The model file"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /models3d/{id}/file [get]
// @Security BearerAuth
func (uc *Model3DUseCase) StreamFile(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, errInvalidModel3DID())
		return
	}

	model, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve 3D model for streaming", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	stream, err := uc.cdnService.StreamFile(ctx, model.FileURL)
	if err != nil {
		if errors.Is(err, services.ErrObjectNotFound) {
			coreErrors.Respond(c, errModel3DNotFound())
			return
		}
		uc.logger.Error(ctx, "Failed to stream 3D model file", map[string]interface{}{"model_id": id, "error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	defer func() { _ = stream.Body.Close() }()

	contentType := services.ContentTypeForFile(model.FileName)

	extraHeaders := map[string]string{
		"Content-Disposition": contentDisposition(model.FileName),
		"Cache-Control":       "private, max-age=300",
	}

	c.DataFromReader(http.StatusOK, stream.Size, contentType, stream.Body, extraHeaders)
}

// contentDisposition builds a safe inline Content-Disposition value for a file
// name. mime.FormatMediaType handles quoting and RFC 2231 encoding for non-ASCII
// names; it returns "" for an unrepresentable name, in which case we fall back to a
// bare "inline".
func contentDisposition(name string) string {
	if d := mime.FormatMediaType("inline", map[string]string{"filename": name}); d != "" {
		return d
	}
	return "inline"
}
