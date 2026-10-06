package usecases

import (
	"bytes"
	"context"
	"crypto/sha256"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/database"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	slicerentities "github.com/RodolfoBonis/spooliq/features/slicer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxFileSize = 50 * 1024 * 1024 // 50MB

// uploadBodyLimit caps the whole request body (file + multipart/metadata overhead)
// so an oversized upload is rejected while streaming, before anything is buffered
// in memory. The slack covers the multipart envelope and the small text fields. It
// is a var (not const) only so tests can shrink it.
var uploadBodyLimit int64 = maxFileSize + (1 << 20) // 50MB + 1MB slack

var allowedExtensions = map[string]bool{
	".stl": true,
	".3mf": true,
}

// Upload handles the multipart upload of a 3D model file.
// @Summary Upload 3D Model
// @Description Upload a new 3D model file (STL or 3MF) with metadata
// @Tags models3d
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "3D model file (.stl or .3mf, max 50MB)"
// @Param name formData string true "Model name"
// @Param description formData string false "Model description"
// @Param customer_id formData string false "Customer ID" format(uuid)
// @Param notes formData string false "Notes"
// @Param tags formData string false "Tags"
// @Success 201 {object} entities.Model3DEntity "Successfully uploaded 3D model"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} entities.DuplicateModel3DResponse "Duplicate file already exists"
// @Failure 413 {object} errors.APIError "File too large"
// @Failure 500 {object} errors.APIError
// @Router /models3d [post]
// @Security BearerAuth
func (uc *Model3DUseCase) Upload(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, errOrganizationRequired())
		return
	}
	userID := helpers.GetUserID(c)

	// Enforce the size limit while STREAMING, before the body is buffered, so an
	// oversized upload can never exhaust memory. MaxBytesReader makes the reader
	// return an error once the cap is exceeded, which surfaces from FormFile below.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, uploadBodyLimit)

	// File part is required.
	fileHeader, err := c.FormFile("file")
	if err != nil {
		if isRequestTooLarge(err) {
			coreErrors.Respond(c, errFileTooLarge())
			return
		}
		coreErrors.Respond(c, errFileRequired())
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		coreErrors.Respond(c, errUnsupportedFileFormat())
		return
	}

	if fileHeader.Size > maxFileSize {
		coreErrors.Respond(c, errFileTooLarge())
		return
	}

	// Bind the multipart form metadata and validate it with the shared validator.
	var request entities.CreateModel3DRequest
	if err := c.ShouldBind(&request); err != nil {
		if isRequestTooLarge(err) {
			coreErrors.Respond(c, errFileTooLarge())
			return
		}
		coreErrors.Respond(c, err)
		return
	}
	if err := validation.Validate(request); err != nil {
		coreErrors.Respond(c, err)
		return
	}

	// Resolve and validate the optional customer reference against the org.
	var customerID *uuid.UUID
	if request.CustomerID != nil && *request.CustomerID != "" {
		parsed, err := uuid.Parse(*request.CustomerID)
		if err != nil {
			coreErrors.Respond(c, errInvalidCustomerID())
			return
		}
		exists, err := uc.repository.CustomerExists(ctx, parsed, organizationID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to validate customer", map[string]interface{}{"error": err.Error()})
			coreErrors.Respond(c, err)
			return
		}
		if !exists {
			coreErrors.Respond(c, errCustomerNotFound())
			return
		}
		customerID = &parsed
	}

	// Read the file content once; it is needed for hashing, upload and thumbnailing.
	file, err := fileHeader.Open()
	if err != nil {
		uc.logger.Error(ctx, "Failed to open uploaded file", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	defer func() { _ = file.Close() }()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		if isRequestTooLarge(err) {
			coreErrors.Respond(c, errFileTooLarge())
			return
		}
		uc.logger.Error(ctx, "Failed to read file content", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	fileBytes := buf.Bytes()

	hash := fmt.Sprintf("%x", sha256.Sum256(fileBytes))

	// Dedup: reject when a model with the same hash already exists in the org.
	existing, err := uc.repository.FindByHash(ctx, hash, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check file hash", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}
	if existing != nil {
		respondDuplicate(c, existing)
		return
	}

	// Upload the file to the CDN under a unique key.
	uniqueID := uuid.New().String()
	filename := uniqueID + ext
	fileURL, err := uc.cdnService.UploadFile(ctx, bytes.NewReader(fileBytes), filename, "models3d")
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload file to CDN", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	// Generate a thumbnail best-effort; it is self-contained (timeout, panic-safe,
	// concurrency-bounded) and returns nil on any problem, never blocking the upload.
	var thumbnailURL *string
	if thumbReader := uc.thumbnailService.Generate(ctx, bytes.NewReader(fileBytes), ext); thumbReader != nil {
		thumbFilename := uniqueID + ".png"
		if thumbURL, err := uc.cdnService.UploadFile(ctx, thumbReader, thumbFilename, "models3d_thumbs"); err != nil {
			uc.logger.Warning(ctx, "Thumbnail upload failed", map[string]interface{}{"error": err.Error()})
		} else {
			thumbnailURL = &thumbURL
		}
	}

	// Best-effort slice analysis for sliced 3MF uploads (ext ".3mf" also covers
	// ".gcode.3mf"). The service is self-bounded (timeout, panic-safe,
	// concurrency-gated); any failure is logged and ignored so it never blocks an
	// upload. Suggestions are NOT stored here — they are computed at read time.
	var sliceAnalysis *slicerentities.Analysis
	if ext == ".3mf" {
		analyzeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		if a, aerr := uc.slicerService.Analyze(analyzeCtx, bytes.NewReader(fileBytes), int64(len(fileBytes)), fileHeader.Filename); aerr != nil {
			uc.logger.Info(ctx, "Slice analysis skipped for upload", map[string]interface{}{"error": aerr.Error()})
		} else {
			sliceAnalysis = a
		}
		cancel()
	}

	model := &entities.Model3DEntity{
		OrganizationID: organizationID,
		CustomerID:     customerID,
		Name:           request.Name,
		Description:    request.Description,
		FileName:       sanitizeFileName(fileHeader.Filename),
		FileURL:        fileURL,
		FileFormat:     ext,
		FileSizeBytes:  fileHeader.Size,
		FileHash:       hash,
		ThumbnailURL:   thumbnailURL,
		Notes:          request.Notes,
		Tags:           request.Tags,
		OwnerUserID:    userID,
		SliceAnalysis:  sliceAnalysis,
	}

	if err := uc.repository.Create(ctx, model); err != nil {
		// A concurrent upload of the same content can race past the FindByHash
		// check and trip the (organization_id, file_hash) partial unique index.
		// Map that to the same 409 by re-reading the now-present row.
		if database.IsUniqueViolation(err) {
			if dup, derr := uc.repository.FindByHash(ctx, hash, organizationID); derr == nil && dup != nil {
				respondDuplicate(c, dup)
				return
			}
		}
		uc.logger.Error(ctx, "Failed to save 3D model", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	uc.logger.Info(ctx, "3D model uploaded successfully", map[string]interface{}{"model_id": model.ID, "model_name": model.Name})

	c.JSON(http.StatusCreated, model)

	uc.activityService.Record(ctx, activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityModel3D,
		EntityID:       model.ID.String(),
		EntityName:     model.Name,
		Description:    "3D Model created: " + model.Name,
	})
}

// isRequestTooLarge reports whether err is (or wraps) the MaxBytesReader limit
// error. Go returns a typed *http.MaxBytesError, but multipart parsing may wrap it
// as a plain message, so we also match the well-known string.
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

// sanitizeFileName makes a user-supplied file name safe to echo back in a
// Content-Disposition header and to store: it strips any path, control characters,
// quotes and backslashes, and trims surrounding whitespace, while keeping the
// extension. Empty results fall back to a generic name.
func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f { // control chars (incl. CR/LF/TAB)
			return -1
		}
		switch r {
		case '"', '\\', '/':
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "model"
	}
	return name
}
