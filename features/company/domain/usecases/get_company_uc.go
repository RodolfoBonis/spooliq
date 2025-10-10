package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/company/domain/entities"
	"github.com/gin-gonic/gin"
)

// Get retrieves the company for the current organization
// @Summary Get company
// @Description Get company information for the current organization
// @Tags company
// @Accept json
// @Produce json
// @Success 200 {object} entities.CompanyResponse
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/company [get]
// @Security BearerAuth
func (uc *CompanyUseCase) Get(c *gin.Context) {
	ctx := c.Request.Context()

	uc.logger.Info(ctx, "Company retrieval attempt started", map[string]interface{}{
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

	company, err := uc.repository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		if err == entities.ErrCompanyNotFound {
			uc.logger.Info(ctx, "Company not found", map[string]interface{}{
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

	uc.logger.Info(ctx, "Company retrieved successfully", map[string]interface{}{
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
