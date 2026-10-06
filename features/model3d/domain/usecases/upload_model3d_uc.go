package usecases

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/database"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxFileSize = 50 * 1024 * 1024 // 50MB

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

	// File part is required.
	fileHeader, err := c.FormFile("file")
	if err != nil {
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

	// Generate a thumbnail best-effort; failures never block the upload.
	var thumbnailURL *string
	thumbReader, err := uc.thumbnailService.Generate(bytes.NewReader(fileBytes), ext)
	if err != nil {
		uc.logger.Warning(ctx, "Thumbnail generation failed", map[string]interface{}{"error": err.Error(), "filename": fileHeader.Filename})
	}
	if thumbReader != nil {
		thumbFilename := uniqueID + ".png"
		if thumbURL, err := uc.cdnService.UploadFile(ctx, thumbReader, thumbFilename, "models3d_thumbs"); err != nil {
			uc.logger.Warning(ctx, "Thumbnail upload failed", map[string]interface{}{"error": err.Error()})
		} else {
			thumbnailURL = &thumbURL
		}
	}

	model := &entities.Model3DEntity{
		OrganizationID: organizationID,
		CustomerID:     customerID,
		Name:           request.Name,
		Description:    request.Description,
		FileName:       fileHeader.Filename,
		FileURL:        fileURL,
		FileFormat:     ext,
		FileSizeBytes:  fileHeader.Size,
		FileHash:       hash,
		ThumbnailURL:   thumbnailURL,
		Notes:          request.Notes,
		Tags:           request.Tags,
		OwnerUserID:    userID,
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
