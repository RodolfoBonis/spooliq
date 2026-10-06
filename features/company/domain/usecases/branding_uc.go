package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/company/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// IBrandingUseCase defines the interface for branding use cases
type IBrandingUseCase interface {
	GetBranding(c *gin.Context)
	UpdateBranding(c *gin.Context)
	ListTemplates(c *gin.Context)
}

// BrandingUseCase implements the branding use cases
type BrandingUseCase struct {
	repository repositories.BrandingRepository
	logger     logger.Logger
}

// NewBrandingUseCase creates a new instance of BrandingUseCase
func NewBrandingUseCase(
	repository repositories.BrandingRepository,
	logger logger.Logger,
) IBrandingUseCase {
	return &BrandingUseCase{
		repository: repository,
		logger:     logger,
	}
}

// GetBranding godoc
// @Summary Get company branding configuration
// @Description Retrieves the current branding configuration for the company, or returns the default template if not configured
// @Tags Company Branding
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.CompanyBrandingEntity
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /company/branding [get]
func (uc *BrandingUseCase) GetBranding(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		errors.Respond(c, errors.BadRequest("organization_id_missing", "Organização não encontrada no contexto"))
		return
	}

	uc.logger.Info(ctx, "Get branding attempt started", map[string]interface{}{
		"organization_id": organizationID,
	})

	// Try to get existing branding
	branding, err := uc.repository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		// If not found, return default template
		uc.logger.Info(ctx, "No custom branding found, returning default template", map[string]interface{}{
			"organization_id": organizationID,
		})
		defaultBranding := entities.GetDefaultTemplate()
		defaultBranding.OrganizationID = organizationID
		c.JSON(http.StatusOK, gin.H{"branding": defaultBranding})
		return
	}

	uc.logger.Info(ctx, "Branding retrieved successfully", map[string]interface{}{
		"organization_id": organizationID,
		"template_name":   branding.TemplateName,
	})

	c.JSON(http.StatusOK, gin.H{"branding": branding})
}

// UpdateBrandingRequest represents the request body for updating branding
type UpdateBrandingRequest struct {
	TemplateName       string `json:"template_name"`
	HeaderBgColor      string `json:"header_bg_color" validate:"required,hexcolor"`
	HeaderTextColor    string `json:"header_text_color" validate:"required,hexcolor"`
	PrimaryColor       string `json:"primary_color" validate:"required,hexcolor"`
	PrimaryTextColor   string `json:"primary_text_color" validate:"required,hexcolor"`
	SecondaryColor     string `json:"secondary_color" validate:"required,hexcolor"`
	SecondaryTextColor string `json:"secondary_text_color" validate:"required,hexcolor"`
	TitleColor         string `json:"title_color" validate:"required,hexcolor"`
	BodyTextColor      string `json:"body_text_color" validate:"required,hexcolor"`
	AccentColor        string `json:"accent_color" validate:"required,hexcolor"`
	BorderColor        string `json:"border_color" validate:"required,hexcolor"`
	BackgroundColor    string `json:"background_color" validate:"required,hexcolor"`
	TableHeaderBgColor string `json:"table_header_bg_color" validate:"required,hexcolor"`
	TableRowAltBgColor string `json:"table_row_alt_bg_color" validate:"required,hexcolor"`
}

// UpdateBranding godoc
// @Summary Update company branding configuration
// @Description Updates or creates the branding configuration for the company with custom colors
// @Tags Company Branding
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param branding body UpdateBrandingRequest true "Branding configuration"
// @Success 200 {object} entities.CompanyBrandingEntity
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /company/branding [put]
func (uc *BrandingUseCase) UpdateBranding(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		errors.Respond(c, errors.BadRequest("organization_id_missing", "Organização não encontrada no contexto"))
		return
	}

	uc.logger.Info(ctx, "Update branding attempt started", map[string]interface{}{
		"organization_id": organizationID,
	})

	var request UpdateBrandingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind request", map[string]interface{}{
			"error": err.Error(),
		})
		errors.Respond(c, err)
		return
	}

	// Validate request (pt-BR field errors via the shared validator)
	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		errors.Respond(c, err)
		return
	}

	// Check if branding already exists
	existingBranding, err := uc.repository.FindByOrganizationID(ctx, organizationID)

	branding := &entities.CompanyBrandingEntity{
		OrganizationID:     organizationID,
		TemplateName:       request.TemplateName,
		HeaderBgColor:      request.HeaderBgColor,
		HeaderTextColor:    request.HeaderTextColor,
		PrimaryColor:       request.PrimaryColor,
		PrimaryTextColor:   request.PrimaryTextColor,
		SecondaryColor:     request.SecondaryColor,
		SecondaryTextColor: request.SecondaryTextColor,
		TitleColor:         request.TitleColor,
		BodyTextColor:      request.BodyTextColor,
		AccentColor:        request.AccentColor,
		BorderColor:        request.BorderColor,
		BackgroundColor:    request.BackgroundColor,
		TableHeaderBgColor: request.TableHeaderBgColor,
		TableRowAltBgColor: request.TableRowAltBgColor,
	}

	if err != nil {
		// Create new branding
		branding.ID = uuid.New()
		if err := uc.repository.Create(ctx, branding); err != nil {
			uc.logger.Error(ctx, "Failed to create branding", map[string]interface{}{
				"error": err.Error(),
			})
			errors.Respond(c, err)
			return
		}

		uc.logger.Info(ctx, "Branding created successfully", map[string]interface{}{
			"organization_id": organizationID,
			"template_name":   branding.TemplateName,
		})
	} else {
		// Update existing branding
		branding.ID = existingBranding.ID
		if err := uc.repository.Update(ctx, branding); err != nil {
			uc.logger.Error(ctx, "Failed to update branding", map[string]interface{}{
				"error": err.Error(),
			})
			errors.Respond(c, err)
			return
		}

		uc.logger.Info(ctx, "Branding updated successfully", map[string]interface{}{
			"organization_id": organizationID,
			"template_name":   branding.TemplateName,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Branding configuration saved successfully",
		"branding": branding,
	})
}

// ListTemplates godoc
// @Summary List available branding templates
// @Description Returns all pre-defined branding templates that users can choose from
// @Tags Company Branding
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 20, max 100)"
// @Success 200 {object} helpers.Page[entities.BrandingTemplate] "Paginated branding templates"
// @Failure 401 {object} errors.HTTPError "Unauthorized"
// @Router /company/branding/templates [get]
func (uc *BrandingUseCase) ListTemplates(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "List templates request", nil)

	templates := uc.repository.GetTemplates()

	// Branding templates are a small, static catalog; paginate in memory so the
	// response still uses the standard envelope.
	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{DefaultPageSize: 20})
	total := int64(len(templates))
	off := q.Offset()
	if off > len(templates) {
		off = len(templates)
	}
	end := len(templates)
	if q.Limit() > 0 {
		end = off + q.Limit()
		if end > len(templates) {
			end = len(templates)
		}
	}
	c.JSON(http.StatusOK, helpers.NewPage(templates[off:end], total, q))
}
