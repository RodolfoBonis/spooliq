package usecases

import (
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Create creates a new company
// @Summary Create company
// @Description Create a new company for the organization
// @Tags company
// @Accept json
// @Produce json
// @Param request body entities.CreateCompanyRequest true "Create company request"
// @Success 201 {object} entities.CompanyResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/company [post]
// @Security BearerAuth
func (uc *CompanyUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Company creation attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Check if user is Platform Admin
	isPlatformAdmin := helpers.IsPlatformAdmin(c)

	// Get organization ID from context
	organizationID := getOrganizationID(c)

	// Platform Admin can create companies without organization_id in context
	// Regular users MUST have organization_id from JWT
	if organizationID == "" && !isPlatformAdmin {
		uc.logger.Error(ctx, "Organization ID not found in context", map[string]interface{}{
			"is_platform_admin": isPlatformAdmin,
		})
		appError := coreErrors.UsecaseError("Organization ID required")
		c.JSON(http.StatusForbidden, gin.H{"error": appError.Message})
		return
	}

	var request entities.CreateCompanyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid company creation payload", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Invalid request format")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		uc.logger.Error(ctx, "Company creation validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.UsecaseError("Validation failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Determine target organization ID
	// Platform Admin: use own org_id OR generate new UUID for client
	// Regular users: use their own org_id from JWT
	targetOrgID := organizationID
	if isPlatformAdmin {
		// If Platform Admin doesn't have org_id in JWT, generate a new one
		// This UUID will need to be added to Keycloak for the client user
		if targetOrgID == "" {
			targetOrgID = uuid.New().String()
			uc.logger.Info(ctx, "Platform Admin creating company with new organization_id", map[string]interface{}{
				"generated_organization_id": targetOrgID,
			})
		} else {
			uc.logger.Info(ctx, "Platform Admin creating company for own organization", map[string]interface{}{
				"organization_id": targetOrgID,
			})
		}
	}

	// Check if company already exists for this organization
	exists, err := uc.repository.ExistsByOrganizationID(ctx, targetOrgID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check company existence", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": targetOrgID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	if exists {
		uc.logger.Error(ctx, "Company already exists for this organization", map[string]interface{}{
			"organization_id": targetOrgID,
		})
		appError := coreErrors.UsecaseError(entities.ErrCompanyAlreadyExists.Error())
		c.JSON(http.StatusConflict, gin.H{"error": appError.Message})
		return
	}

	company := &entities.CompanyEntity{
		ID:             uuid.New(),
		OrganizationID: targetOrgID,
		Name:           request.Name,
		TradeName:      request.TradeName,
		Document:       request.Document,
		Email:          request.Email,
		Phone:          request.Phone,
		WhatsApp:       request.WhatsApp,
		Instagram:      request.Instagram,
		Website:        request.Website,
		LogoURL:        request.LogoURL,
		Address:        request.Address,
		City:           request.City,
		State:          request.State,
		ZipCode:        request.ZipCode,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := uc.repository.Create(ctx, company); err != nil {
		uc.logger.Error(ctx, "Failed to create company", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	uc.logger.Info(ctx, "Company created successfully", map[string]interface{}{
		"company_id":      company.ID,
		"company_name":    company.Name,
		"organization_id": company.OrganizationID,
	})

	c.JSON(http.StatusCreated, entities.CompanyResponse{
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
