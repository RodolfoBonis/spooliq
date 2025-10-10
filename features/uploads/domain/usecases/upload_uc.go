package usecases

import (
	"net/http"
	"path/filepath"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IUploadUseCase defines the interface for upload use cases
type IUploadUseCase interface {
	UploadLogo(c *gin.Context)
	UploadFile(c *gin.Context)
}

// UploadUseCase implements the upload use cases
type UploadUseCase struct {
	cdnService *services.CDNService
	logger     logger.Logger
}

// NewUploadUseCase creates a new instance of UploadUseCase
func NewUploadUseCase(cdnService *services.CDNService, logger logger.Logger) IUploadUseCase {
	return &UploadUseCase{
		cdnService: cdnService,
		logger:     logger,
	}
}

// UploadLogo uploads a company logo
// @Summary Upload company logo
// @Description Upload a logo image for the company
// @Tags uploads
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Logo file"
// @Success 200 {object} map[string]string{"url": "string"}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/uploads/logo [post]
// @Security BearerAuth
func (uc *UploadUseCase) UploadLogo(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Logo upload attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		uc.logger.Error(ctx, "Failed to get file from request", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("File is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}
	defer file.Close()

	// Validate file type
	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".svg":  true,
	}

	if !allowedExts[ext] {
		uc.logger.Error(ctx, "Invalid file type", map[string]interface{}{
			"extension": ext,
			"filename":  header.Filename,
		})
		appError := coreErrors.UsecaseError("Invalid file type. Allowed: jpg, jpeg, png, webp, svg")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		uc.logger.Error(ctx, "File too large", map[string]interface{}{
			"size":     header.Size,
			"filename": header.Filename,
		})
		appError := coreErrors.UsecaseError("File size exceeds 5MB limit")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + ext

	// Upload to CDN
	url, err := uc.cdnService.UploadFile(ctx, file, filename, "logos")
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload logo to CDN", map[string]interface{}{
			"error":    err.Error(),
			"filename": filename,
		})
		appError := coreErrors.UsecaseError("Failed to upload file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Logo uploaded successfully", map[string]interface{}{
		"url":      url,
		"filename": filename,
	})

	c.JSON(http.StatusOK, gin.H{
		"url":     url,
		"message": "Logo uploaded successfully",
	})
}

// UploadFile uploads a generic file
// @Summary Upload file
// @Description Upload a generic file (PDF, 3MF, images, etc.)
// @Tags uploads
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Param folder formData string false "Optional folder path"
// @Success 200 {object} map[string]string{"url": "string"}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/uploads/file [post]
// @Security BearerAuth
func (uc *UploadUseCase) UploadFile(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "File upload attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Get file from request
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		uc.logger.Error(ctx, "Failed to get file from request", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("File is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}
	defer file.Close()

	// Validate file type
	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{
		".jpg":   true,
		".jpeg":  true,
		".png":   true,
		".webp":  true,
		".svg":   true,
		".pdf":   true,
		".3mf":   true,
		".stl":   true,
		".gcode": true,
	}

	if !allowedExts[ext] {
		uc.logger.Error(ctx, "Invalid file type", map[string]interface{}{
			"extension": ext,
			"filename":  header.Filename,
		})
		appError := coreErrors.UsecaseError("Invalid file type. Allowed: jpg, jpeg, png, webp, svg, pdf, 3mf, stl, gcode")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate file size (max 50MB)
	if header.Size > 50*1024*1024 {
		uc.logger.Error(ctx, "File too large", map[string]interface{}{
			"size":     header.Size,
			"filename": header.Filename,
		})
		appError := coreErrors.UsecaseError("File size exceeds 50MB limit")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Get optional folder
	folder := c.PostForm("folder")
	if folder == "" {
		folder = "files"
	}

	// Generate unique filename
	filename := uuid.New().String() + ext

	// Upload to CDN
	url, err := uc.cdnService.UploadFile(ctx, file, filename, folder)
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload file to CDN", map[string]interface{}{
			"error":    err.Error(),
			"filename": filename,
		})
		appError := coreErrors.UsecaseError("Failed to upload file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "File uploaded successfully", map[string]interface{}{
		"url":      url,
		"filename": filename,
		"folder":   folder,
	})

	c.JSON(http.StatusOK, gin.H{
		"url":     url,
		"message": "File uploaded successfully",
	})
}
