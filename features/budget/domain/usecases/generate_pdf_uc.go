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

	// Build items response with filaments and calculate total print time
	itemsResponse := make([]entities.BudgetItemResponse, 0, len(items))
	var totalPrintMinutes int

	for _, item := range items {
		// Get filament usage info for this item
		filaments, err := uc.budgetRepository.GetFilamentUsageInfo(ctx, item.ID)
		if err != nil {
			uc.logger.Error(ctx, "Failed to get filament usage info", map[string]interface{}{
				"error":   err.Error(),
				"item_id": item.ID,
			})
		}

		// Calculate print time display
		printTimeDisplay := ""
		if item.PrintTimeHours > 0 {
			printTimeDisplay = fmt.Sprintf("%dh%02dm", item.PrintTimeHours, item.PrintTimeMinutes)
		} else {
			printTimeDisplay = fmt.Sprintf("%dm", item.PrintTimeMinutes)
		}

		// Sum total print time
		totalPrintMinutes += (item.PrintTimeHours * 60) + item.PrintTimeMinutes

		// Convert CostPresetID to string pointer
		var costPresetIDStr *string
		if item.CostPresetID != nil {
			s := item.CostPresetID.String()
			costPresetIDStr = &s
		}

		itemsResponse = append(itemsResponse, entities.BudgetItemResponse{
			ID:                  item.ID.String(),
			BudgetID:            item.BudgetID.String(),
			ProductName:         item.ProductName,
			ProductDescription:  item.ProductDescription,
			ProductQuantity:     item.ProductQuantity,
			ProductDimensions:   item.ProductDimensions,
			PrintTimeHours:      item.PrintTimeHours,
			PrintTimeMinutes:    item.PrintTimeMinutes,
			PrintTimeDisplay:    printTimeDisplay,
			CostPresetID:        costPresetIDStr,
			AdditionalLaborCost: item.AdditionalLaborCost,
			AdditionalNotes:     item.AdditionalNotes,
			FilamentCost:        item.FilamentCost,
			WasteCost:           item.WasteCost,
			EnergyCost:          item.EnergyCost,
			LaborCost:           item.LaborCost,
			ItemTotalCost:       item.ItemTotalCost,
			UnitPrice:           item.UnitPrice,
			Filaments:           filaments,
			Order:               item.Order,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		})
	}

	// Calculate total print time
	totalHours := totalPrintMinutes / 60
	totalMins := totalPrintMinutes % 60

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
