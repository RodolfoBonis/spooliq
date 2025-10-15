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
		"user_agent":      c.Request.UserAgent(),
		"ip":              c.ClientIP(),
		"organization_id": organizationID,
	})

	// Get file from form data
	file, err := c.FormFile("logo")
	if err != nil {
		uc.logger.Error(ctx, "Failed to get logo file", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Logo file is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
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
			"filename":  file.Filename,
		})
		appError := coreErrors.UsecaseError("Only PNG, JPG, and JPEG files are allowed")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Validate file size (max 5MB)
	maxSize := int64(5 * 1024 * 1024) // 5MB
	if file.Size > maxSize {
		uc.logger.Error(ctx, "File size too large", map[string]interface{}{
			"size":     file.Size,
			"max_size": maxSize,
		})
		appError := coreErrors.UsecaseError("Logo file must be less than 5MB")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Open file
	fileContent, err := file.Open()
	if err != nil {
		uc.logger.Error(ctx, "Failed to open file", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Failed to process logo file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
		return
	}
	defer fileContent.Close()

	// Upload to CDN
	filename := fmt.Sprintf("logo%s", ext)
	folder := fmt.Sprintf("org-%s/company", organizationID)

	cdnURL, err := uc.cdnService.UploadFile(ctx, fileContent, filename, folder)
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
		"filename":   file.Filename,
		"size":       file.Size,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":  "Logo uploaded successfully",
		"logo_url": cdnURL,
	})
}
