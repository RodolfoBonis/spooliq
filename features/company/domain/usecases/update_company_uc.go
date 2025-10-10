package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/gin-gonic/gin"
)

// Update updates the company information
// @Summary Update company
// @Description Update company information for the current organization
// @Tags company
// @Accept json
// @Produce json
// @Param request body entities.UpdateCompanyRequest true "Update company request"
// @Success 200 {object} entities.CompanyResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/company [put]
// @Security BearerAuth
func (uc *CompanyUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Company update attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	organizationID := getOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		appError := coreErrors.UsecaseError("Organization ID not found in context")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Find existing company
	company, err := uc.repository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		if err == entities.ErrCompanyNotFound {
			uc.logger.Error(ctx, "Company not found", map[string]interface{}{
				"organization_id": organizationID,
			})
			appError := coreErrors.UsecaseError(err.Error())
			c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
			return
		}

		uc.logger.Error(ctx, "Failed to retrieve company", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	var request entities.UpdateCompanyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid company update payload", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid request format")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		uc.logger.Error(ctx, "Company update validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Validation failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Update fields if provided
	if request.Name != nil {
		company.Name = *request.Name
	}
	if request.TradeName != nil {
		company.TradeName = request.TradeName
	}
	if request.Document != nil {
		company.Document = request.Document
	}
	if request.Email != nil {
		company.Email = request.Email
	}
	if request.Phone != nil {
		company.Phone = request.Phone
	}
	if request.WhatsApp != nil {
		company.WhatsApp = request.WhatsApp
	}
	if request.Instagram != nil {
		company.Instagram = request.Instagram
	}
	if request.Website != nil {
		company.Website = request.Website
	}
	if request.LogoURL != nil {
		company.LogoURL = request.LogoURL
	}
	if request.Address != nil {
		company.Address = request.Address
	}
	if request.City != nil {
		company.City = request.City
	}
	if request.State != nil {
		company.State = request.State
	}
	if request.ZipCode != nil {
		company.ZipCode = request.ZipCode
	}

	company.UpdatedAt = time.Now()

	if err := uc.repository.Update(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to update company", map[string]interface{}{
			"error":      err.Error(),
			"company_id": company.ID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Company updated successfully", map[string]interface{}{
		"company_id":      company.ID,
		"organization_id": company.OrganizationID,
	})

	c.JSON(http.StatusOK, entities.CompanyResponse{
		ID:             company.ID.String(),
		OrganizationID: company.OrganizationID,
		Name:           company.Name,
		TradeName:      company.TradeName,
		Document:       company.Document,
		Email:          company.Email,
		Phone:          company.Phone,
		WhatsApp:       company.WhatsApp,
		Instagram:      company.Instagram,
		Website:        company.Website,
		LogoURL:        company.LogoURL,
		Address:        company.Address,
		City:           company.City,
		State:          company.State,
		ZipCode:        company.ZipCode,
		CreatedAt:      company.CreatedAt,
		UpdatedAt:      company.UpdatedAt,
	})
}
