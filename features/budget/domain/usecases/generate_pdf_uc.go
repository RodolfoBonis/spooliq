package usecases

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/services"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GeneratePDF generates a PDF for a budget
// @Summary Generate budget PDF
// @Description Generate and download PDF for a specific budget
// @Tags budgets
// @Accept json
// @Produce application/pdf
// @Param id path string true "Budget ID"
// @Success 200 {file} application/pdf
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/{id}/pdf [get]
// @Security BearerAuth
func (uc *BudgetUseCase) GeneratePDF(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	uc.logger.Info(ctx, "PDF generation attempt started", map[string]interface{}{
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	})

	// Get budget ID from path
	budgetIDStr := c.Param("id")
	budgetID, err := uuid.Parse(budgetIDStr)
	if err != nil {
		uc.logger.Error(ctx, "Invalid budget ID format", map[string]interface{}{
			"budget_id": budgetIDStr,
		})
		appError := coreErrors.UsecaseError("Invalid budget ID format")
		c.JSON(http.StatusBadRequest, gin.H{"error": appError.Message})
		return
	}

	// Get budget
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Budget not found", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		appError := coreErrors.UsecaseError("Budget not found")
		c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
		return
	}

	// Get customer info
	customer, err := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get customer info", map[string]interface{}{
			"error":       err.Error(),
			"customer_id": budget.CustomerID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Get budget items with filament info
	items, err := uc.budgetRepository.FindItemsByBudgetID(ctx, budgetID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get budget items", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		appError := coreErrors.RepositoryError(err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build items response with filament info
	itemsResponse := make([]entities.BudgetItemResponse, 0, len(items))
	for _, item := range items {
		filamentInfo, err := uc.budgetRepository.GetFilamentInfo(ctx, item.FilamentID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to get filament info", map[string]interface{}{
				"error":       err.Error(),
				"filament_id": item.FilamentID,
			})
			continue
		}

		itemsResponse = append(itemsResponse, entities.BudgetItemResponse{
			ID:          item.ID.String(),
			BudgetID:    item.BudgetID.String(),
			FilamentID:  item.FilamentID.String(),
			Filament:    filamentInfo,
			Quantity:    item.Quantity,
			Order:       item.Order,
			WasteAmount: item.WasteAmount,
			ItemCost:    item.ItemCost,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	// Get company info
	company, err := uc.budgetRepository.GetCompanyByOrganizationID(ctx, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to get company info", map[string]interface{}{
			"error":           err.Error(),
			"organization_id": organizationID,
		})
		appError := coreErrors.RepositoryError("Company information not found. Please configure your company settings first.")
		c.JSON(http.StatusNotFound, gin.H{"error": appError.Message})
		return
	}

	// Generate PDF
	pdfData := services.BudgetPDFData{
		Budget:   budget,
		Customer: customer,
		Items:    itemsResponse,
		Company:  company,
	}

	pdfService := uc.pdfService
	pdfBytes, err := pdfService.GenerateBudgetPDF(ctx, pdfData)
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate PDF", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		appError := coreErrors.UsecaseError("Failed to generate PDF")
		c.JSON(http.StatusInternalServerError, gin.H{"error": appError.Message})
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
		// Save CDN URL to database
		budget.PDFUrl = &cdnURL
		err = uc.budgetRepository.Update(ctx, budget)
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

	// Return PDF
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
