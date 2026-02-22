package usecases

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
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
// Schemes
// @Description Upload a new 3D model file (STL or 3MF) with metadata
// @Tags 3D Models
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "3D model file (.stl or .3mf, max 50MB)"
// @Param name formData string true "Model name"
// @Param description formData string false "Model description"
// @Param customer_id formData string false "Customer ID" format(uuid)
// @Param notes formData string false "Notes"
// @Param tags formData string false "Tags"
// @Success 201 {object} entities.Model3DEntity "Successfully uploaded 3D model"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} object "Duplicate file already exists"
// @Failure 500 {object} errors.HTTPError
// @Router /models3d [post]
// @Security Bearer
func (uc *Model3DUseCase) Upload(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return
	}

	userID := helpers.GetUserID(c)

	// Parse multipart file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		uc.logger.Error(ctx, "Failed to read uploaded file", map[string]interface{}{
			"error": err.Error(),
		})
		appError := errors.UsecaseError("File is required")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		appError := errors.UsecaseError("Unsupported file format. Only .stl and .3mf are allowed")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Validate file size
	if fileHeader.Size > maxFileSize {
		appError := errors.UsecaseError("File size exceeds the 50MB limit")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Read form fields
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		appError := errors.UsecaseError("Name is required")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}
	if len(name) > 255 {
		appError := errors.UsecaseError("Name must not exceed 255 characters")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if len(fileHeader.Filename) > 255 {
		appError := errors.UsecaseError("File name must not exceed 255 characters")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	description := c.PostForm("description")
	if len(description) > 2000 {
		appError := errors.UsecaseError("Description must not exceed 2000 characters")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	customerIDStr := c.PostForm("customer_id")
	tags := c.PostForm("tags")
	if len(tags) > 1000 {
		appError := errors.UsecaseError("Tags must not exceed 1000 characters")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	notes := c.PostForm("notes")
	if len(notes) > 2000 {
		appError := errors.UsecaseError("Notes must not exceed 2000 characters")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Open the file and read its content
	file, err := fileHeader.Open()
	if err != nil {
		uc.logger.Error(ctx, "Failed to open uploaded file", map[string]interface{}{
			"error": err.Error(),
		})
		appError := errors.UsecaseError("Failed to process uploaded file")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		uc.logger.Error(ctx, "Failed to read file content", map[string]interface{}{
			"error": err.Error(),
		})
		appError := errors.UsecaseError("Failed to read uploaded file")
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}
	fileBytes := buf.Bytes()

	// Compute SHA-256 hash
	hash := fmt.Sprintf("%x", sha256.Sum256(fileBytes))

	// Dedup: check if a model with the same hash already exists
	existing, err := uc.repository.FindByHash(hash, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check file hash", map[string]interface{}{
			"error": err.Error(),
		})
		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to check for duplicate file")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":    "A file with the same content already exists",
			"existing": existing,
		})
		return
	}

	// Parse customer_id if provided
	var customerID *uuid.UUID
	if customerIDStr != "" {
		parsed, err := uuid.Parse(customerIDStr)
		if err != nil {
			appError := errors.UsecaseError("Invalid customer_id format")
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
		customerID = &parsed
	}

	// Generate unique filename and upload to CDN
	uniqueID := uuid.New().String()
	filename := uniqueID + ext
	fileURL, err := uc.cdnService.UploadFile(ctx, bytes.NewReader(fileBytes), filename, "models3d")
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload file to CDN", map[string]interface{}{
			"error": err.Error(),
		})
		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to upload file")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Generate thumbnail (non-blocking on failure)
	var thumbnailURL *string
	thumbReader, err := uc.thumbnailService.Generate(bytes.NewReader(fileBytes), ext)
	if err != nil {
		uc.logger.Warning(ctx, "Thumbnail generation failed", map[string]interface{}{
			"error":    err.Error(),
			"filename": fileHeader.Filename,
		})
	}
	if thumbReader != nil {
		thumbFilename := uniqueID + ".png"
		thumbURL, err := uc.cdnService.UploadFile(ctx, thumbReader, thumbFilename, "models3d_thumbs")
		if err != nil {
			uc.logger.Warning(ctx, "Thumbnail upload failed", map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			thumbnailURL = &thumbURL
		}
	}

	// Build optional string pointers
	var tagsPtr, notesPtr *string
	if tags != "" {
		tagsPtr = &tags
	}
	if notes != "" {
		notesPtr = &notes
	}

	model := &entities.Model3DEntity{
		OrganizationID: organizationID,
		CustomerID:     customerID,
		Name:           name,
		Description:    description,
		FileName:       fileHeader.Filename,
		FileURL:        fileURL,
		FileFormat:     ext,
		FileSizeBytes:  fileHeader.Size,
		FileHash:       hash,
		ThumbnailURL:   thumbnailURL,
		Notes:          notesPtr,
		Tags:           tagsPtr,
		OwnerUserID:    userID,
	}

	if err := uc.repository.Create(model); err != nil {
		uc.logger.Error(ctx, "Failed to save 3D model", map[string]interface{}{
			"error": err.Error(),
		})
		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to save 3D model")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(ctx, "3D model uploaded successfully", map[string]interface{}{
		"model_id":   model.ID,
		"model_name": model.Name,
	})

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
