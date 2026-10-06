package usecases

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GeneratePDF generates a PDF for a budget
// @Summary Generate budget PDF
// @Description Generate PDF for a specific budget and return CDN URL. Use ?force=true to regenerate existing PDF.
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param force query bool false "Force regenerate PDF even if exists"
// @Success 200 {object} map[string]interface{} "PDF URL and metadata"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/pdf [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GeneratePDF(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	// Get budget ID from path
	budgetIDStr := c.Param("id")
	budgetID, err := uuid.Parse(budgetIDStr)
	if err != nil {
		uc.logger.Error(ctx, "Invalid budget ID format", map[string]interface{}{"budget_id": budgetIDStr})
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	// Check if force regeneration is requested
	forceRegenerate := c.Query("force") == "true"

	// Get budget
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Budget not found", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		respondBudgetError(c, err)
		return
	}

	// Check if PDF already exists and force regeneration is not requested
	if budget.PDFUrl != nil && *budget.PDFUrl != "" && !forceRegenerate {
		uc.logger.Info(ctx, "Returning existing PDF URL", map[string]interface{}{
			"budget_id": budgetID,
			"pdf_url":   *budget.PDFUrl,
		})

		c.JSON(http.StatusOK, gin.H{
			"pdf_url":     *budget.PDFUrl,
			"budget_id":   budgetID.String(),
			"budget_name": budget.Name,
			"generated":   false,
			"message":     "Existing PDF returned. Use ?force=true to regenerate.",
		})
		return
	}

	// Get customer info
	customer, err := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get customer info", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": budget.CustomerID,
		})
		respondBudgetError(c, err)
		return
	}

	// Get budget items with filament info
	items, err := uc.budgetRepository.FindItemsByBudgetID(ctx, budgetID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get budget items", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		respondBudgetError(c, err)
		return
	}

	// Build items response (with the shared per-item sale distribution) plus the
	// total print time. Using the same builder as the API guarantees the PDF shows
	// identical per-item sale values.
	itemsResponse, totalHours, totalMins := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.TotalCost, organizationID)

	// Get company info
	company, err := uc.budgetRepository.GetCompanyByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get company info", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		coreErrors.Respond(c, coreErrors.NotFoundErr("company_not_configured", "Informações da empresa não encontradas. Configure os dados da sua empresa primeiro."))
		return
	}

	// Get branding configuration (use default if not found)
	branding, err := uc.brandingRepository.FindByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Info(ctx, "No custom branding found, using default template", map[string]interface{}{
			"organization_id": organizationID,
		})
		// Use default template if branding not found
		branding = nil // PDFService will use default
	}

	// Generate PDF
	pdfData := services.BudgetPDFData{
		Budget:                budget,
		Customer:              customer,
		Items:                 itemsResponse,
		Company:               company,
		Branding:              branding,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
	}

	pdfService := uc.pdfService
	pdfBytes, err := pdfService.GenerateBudgetPDF(ctx, pdfData)
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate PDF", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		coreErrors.Respond(c, coreErrors.Internal())
		return
	}

	uc.logger.Info(ctx, "PDF generated successfully", map[string]interface{}{
		"budget_id": budgetID,
		"size":      len(pdfBytes),
	})

	// Upload PDF to CDN
	filename := fmt.Sprintf("orcamento_%s_%s.pdf", budget.Name, budgetID.String())
	folder := fmt.Sprintf("org-%s/budgets", organizationID)

	pdfReader := bytes.NewReader(pdfBytes)
	cdnURL, err := uc.cdnService.UploadFile(ctx, pdfReader, filename, folder)
	if err != nil {
		uc.logger.Error(ctx, "Failed to upload PDF to CDN", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		// Continue even if CDN upload fails - user can still download the PDF
	} else {
		// Save CDN URL to database via the dedicated pdf_url writer, which is NOT
		// restricted to drafts (PDFs are generated for approved/sent budgets, and
		// the generic Update is draft-only).
		budget.PDFUrl = &cdnURL
		err = uc.budgetRepository.UpdatePDFURL(ctx, budgetID, organizationID, &cdnURL)
		if err != nil {
			uc.logger.Error(ctx, "Failed to save PDF URL to database", map[string]interface{}{
				"error":     err.Error(),
				"budget_id": budgetID,
				"cdn_url":   cdnURL,
			})
			// Continue - PDF is uploaded but URL not saved
		} else {
			uc.logger.Info(ctx, "PDF uploaded to CDN and URL saved", map[string]interface{}{
				"budget_id": budgetID,
				"cdn_url":   cdnURL,
			})
		}
	}

	// Return PDF URL and metadata
	response := gin.H{
		"budget_id":   budgetID.String(),
		"budget_name": budget.Name,
		"generated":   true,
		"message":     "PDF generated successfully",
		"file_size":   len(pdfBytes),
	}

	// Add PDF URL if uploaded to CDN successfully
	if budget.PDFUrl != nil && *budget.PDFUrl != "" {
		response["pdf_url"] = *budget.PDFUrl
		response["cdn_uploaded"] = true
	} else {
		response["cdn_uploaded"] = false
		response["message"] = "PDF generated but CDN upload failed. PDF available for download only."
		// Fallback: still provide the PDF as download if CDN failed
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
		return
	}

	c.JSON(http.StatusOK, response)
}
