package usecases

import (
	stderrors "errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	"github.com/gin-gonic/gin"
)

// maxFileSize is the hard cap on an uploaded sliced file.
const maxFileSize = 200 * 1024 * 1024 // 200MB

// uploadBodyLimit caps the whole request body so an oversized upload is rejected
// while streaming, before anything is buffered. Slack covers the multipart
// envelope. It is a var only so tests can shrink it.
var uploadBodyLimit int64 = maxFileSize + (1 << 20) // 200MB + 1MB slack

// allowedSuffixes are the accepted file name endings. ".gcode.3mf" is listed so a
// double extension is accepted as-is.
var allowedSuffixes = []string{".gcode.3mf", ".gcode", ".3mf"}

func hasAllowedExtension(filename string) bool {
	lower := strings.ToLower(filename)
	for _, s := range allowedSuffixes {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

// Analyze parses an uploaded sliced print file and returns its print time and
// per-color filament usage, enriched with the best-matching org filament per slot.
// @Summary Analyze a sliced print file
// @Description Parses a sliced .gcode, .3mf or .gcode.3mf file and returns print time and per-color filament grams, with suggested organization filaments per slot.
// @Tags Slicer
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Sliced file (.gcode, .3mf or .gcode.3mf, max 200MB)"
// @Success 200 {object} entities.Analysis "Slicer analysis"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 413 {object} errors.APIError "File too large"
// @Failure 422 {object} errors.APIError "File not sliced or analysis failed"
// @Failure 500 {object} errors.APIError
// @Router /slicer/analyze [post]
// @Security BearerAuth
func (uc *SlicerUseCase) Analyze(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}

	// Enforce the size cap while STREAMING, before buffering, so an oversized
	// upload can never exhaust memory.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, uploadBodyLimit)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		if isRequestTooLarge(err) {
			coreErrors.Respond(c, errFileTooLarge())
			return
		}
		coreErrors.Respond(c, errFileRequired())
		return
	}

	if !hasAllowedExtension(fileHeader.Filename) {
		coreErrors.Respond(c, errUnsupportedFileFormat())
		return
	}
	if fileHeader.Size > maxFileSize {
		coreErrors.Respond(c, errFileTooLarge())
		return
	}

	// Stream the upload to a temp file (removed on return) so the parser has a
	// stable io.ReaderAt + size without holding the whole file in memory.
	tmp, size, terr := streamToTempFile(fileHeader)
	if terr != nil {
		if isRequestTooLarge(terr) {
			coreErrors.Respond(c, errFileTooLarge())
			return
		}
		uc.logger.Error(ctx, "Failed to buffer uploaded file", map[string]interface{}{"error": terr.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	// Analyze (bounded: timeout, recover, concurrency gate inside the service).
	analysis, aerr := uc.service.Analyze(ctx, tmp, size, fileHeader.Filename)
	if aerr != nil {
		switch {
		case service.IsNotSliced(aerr):
			coreErrors.Respond(c, errFileNotSliced())
		case service.IsCorruptFile(aerr):
			coreErrors.Respond(c, errInvalidFile())
		case service.IsUnsupported(aerr):
			coreErrors.Respond(c, errUnsupportedFileFormat())
		default:
			uc.logger.Warning(ctx, "Slicer analysis failed", map[string]interface{}{"error": aerr.Error()})
			coreErrors.Respond(c, errSliceAnalysisFailed())
		}
		return
	}

	// Enrich with org filament suggestions (catalog loaded once).
	if serr := uc.service.Suggest(ctx, organizationID, analysis); serr != nil {
		uc.logger.Error(ctx, "Failed to compute filament suggestions", map[string]interface{}{"error": serr.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// streamToTempFile copies the uploaded file to a new temp file and returns it
// (seeked to start) with its size. The caller owns closing and removing it.
func streamToTempFile(fileHeader *multipart.FileHeader) (*os.File, int64, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = src.Close() }()

	tmp, err := os.CreateTemp("", "slicer-*"+tmpSuffix(fileHeader.Filename))
	if err != nil {
		return nil, 0, err
	}

	size, err := io.Copy(tmp, src)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, 0, err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return nil, 0, err
	}
	return tmp, size, nil
}

// tmpSuffix preserves the extension so the parser can classify the temp file.
func tmpSuffix(filename string) string {
	lower := strings.ToLower(filename)
	for _, s := range allowedSuffixes {
		if strings.HasSuffix(lower, s) {
			return s
		}
	}
	return ""
}

// isRequestTooLarge reports whether err is (or wraps) the MaxBytesReader limit.
func isRequestTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxErr *http.MaxBytesError
	if stderrors.As(err, &maxErr) {
		return true
	}
	return strings.Contains(err.Error(), "http: request body too large")
}
