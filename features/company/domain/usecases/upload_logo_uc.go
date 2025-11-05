package usecases

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/gin-gonic/gin"
)

// UploadLogo uploads company logo to CDN
// @Summary Upload company logo
// @Description Upload logo image for the company
// @Tags company
// @Accept multipart/form-data
// @Produce json
// @Param logo formData file true "Logo image file (png, jpg, jpeg)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/company/logo [post]
// @Security BearerAuth
func (uc *CompanyUseCase) UploadLogo(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := getOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "Logo upload attempt started", map[string]interface{}{
		"user_agent":            c.Request.UserAgent(),
		"ip":                    c.ClientIP(),
		"organization_id":       organizationID,
		"content_type":          c.Request.Header.Get("Content-Type"),
		"content_length":        c.Request.ContentLength,
		"method":                c.Request.Method,
		"transfer_encoding":     c.Request.TransferEncoding,
		"host":                  c.Request.Host,
		"proto":                 c.Request.Proto,
		"body_is_nil":           c.Request.Body == nil,
		"multipart_form_is_nil": c.Request.MultipartForm == nil,
	})

	// Check if request is multipart
	if !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		uc.logger.Error(ctx, "Request is not multipart/form-data", map[string]interface{}{
			"content_type": c.Request.Header.Get("Content-Type"),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request must be multipart/form-data"})
		return
	}

	// Get file from form data using c.Request.FormFile (like rb-cdn)
	// Try both "logo" and "file" field names
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		// Try "file" field name as fallback
		file, header, err = c.Request.FormFile("file")
		if err != nil {
			uc.logger.Error(ctx, "Failed to get file from both 'logo' and 'file' fields using c.Request.FormFile", map[string]interface{}{
				"error":          err.Error(),
				"content_type":   c.Request.Header.Get("Content-Type"),
				"content_length": c.Request.ContentLength,
			})

			appError := coreErrors.UsecaseError("Logo file is required or malformed. Please ensure you're sending a valid multipart form with a 'logo' or 'file' field.")
			c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
			return
		}
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := []string{".png", ".jpg", ".jpeg"}
	isValid := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isValid = true
			break
		}
	}

	if !isValid {
		uc.logger.Error(ctx, "Invalid file extension", map[string]interface{}{
			"extension": ext,
			"filename":  header.Filename,
		})
		appError := coreErrors.UsecaseError("Only PNG, JPG, and JPEG files are allowed")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate file size (max 5MB)
	maxSize := int64(5 * 1024 * 1024) // 5MB
	if header.Size > maxSize {
		uc.logger.Error(ctx, "File size too large", map[string]interface{}{
			"size":     header.Size,
			"max_size": maxSize,
		})
		appError := coreErrors.UsecaseError("Logo file must be less than 5MB")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Upload to CDN using the file directly (like rb-cdn)
	filename := fmt.Sprintf("logo%s", ext)
	folder := fmt.Sprintf("org-%s/company", organizationID)

	cdnURL, err := uc.cdnService.UploadFile(ctx, file, filename, folder)
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload logo to CDN", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Failed to upload logo to CDN")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	// Get company
	company, err := uc.repository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := coreErrors.RepositoryError("Company not found")
		c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
		return
	}

	// Update logo URL
	company.LogoURL = &cdnURL
	err = uc.repository.Update(ctx, company)
	if err != nil {
		uc.logger.Error(ctx, "Failed to update company logo URL", map[string]interface{}{
			"error":      err.Error(),
			"cdn_url":    cdnURL,
			"company_id": company.ID,
		})
		appError := coreErrors.RepositoryError("Failed to update company logo")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Logo uploaded successfully", map[string]interface{}{
		"cdn_url":    cdnURL,
		"company_id": company.ID,
		"filename":   header.Filename,
		"size":       header.Size,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "Logo uploaded successfully",
		"logo_url": cdnURL,
	})
}
