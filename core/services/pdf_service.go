package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/jung-kurt/gofpdf/v2"
)

// PDFService handles PDF generation and upload
type PDFService struct {
	cdnService *CDNService
	logger     logger.Logger
}

// BudgetPDFData contains all data needed to generate a budget PDF
type BudgetPDFData struct {
	Budget   *budgetEntities.BudgetEntity
	Customer *budgetEntities.CustomerInfo
	Items    []budgetEntities.BudgetItemResponse
	Company  *budgetEntities.CompanyInfo
}

// NewPDFService creates a new PDF service instance
func NewPDFService(cdnService *CDNService, logger logger.Logger) *PDFService {
	return &PDFService{
		cdnService: cdnService,
		logger:     logger,
	}
}

// GenerateBudgetPDF generates a PDF for a budget
func (s *PDFService) GenerateBudgetPDF(ctx context.Context, data BudgetPDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// Add header
	if err := s.addHeader(pdf, data.Company); err != nil {
		s.logger.Error(ctx, "Failed to add header to PDF", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Add title
	s.addTitle(pdf, data.Budget)

	// Add customer info
	s.addCustomerInfo(pdf, data.Customer)

	// Add items table
	s.addItemsTable(pdf, data.Items)

	// Add cost summary
	s.addCostSummary(pdf, data.Budget)

	// Add additional info (delivery, payment, notes)
	s.addAdditionalInfo(pdf, data.Budget)

	// Add footer
	s.addFooter(pdf, data.Company)

	// Generate PDF bytes
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		s.logger.Error(ctx, "Failed to generate PDF", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// GenerateAndUploadBudgetPDF generates a PDF and uploads it to CDN
func (s *PDFService) GenerateAndUploadBudgetPDF(ctx context.Context, data BudgetPDFData, budgetID string) (string, error) {
	// Generate PDF
	pdfBytes, err := s.GenerateBudgetPDF(ctx, data)
	if err != nil {
		return "", err
	}

	// Upload to CDN
	filename := fmt.Sprintf("budget_%s_%s.pdf", budgetID, time.Now().Format("20060102_150405"))
	reader := bytes.NewReader(pdfBytes)

	url, err := s.cdnService.UploadFile(ctx, reader, filename, "budgets")
	if err != nil {
		s.logger.Error(ctx, "Failed to upload PDF to CDN", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		return "", fmt.Errorf("failed to upload PDF: %w", err)
	}

	s.logger.Info(ctx, "Budget PDF generated and uploaded successfully", map[string]interface{}{
		"budget_id": budgetID,
		"url":       url,
	})

	return url, nil
}

// addHeader adds the company header with logo
func (s *PDFService) addHeader(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo) error {
	// Pink/rose theme
	pdf.SetFillColor(255, 192, 203) // Light pink

	// Company logo (if available)
	if company.LogoURL != nil && *company.LogoURL != "" {
		// Try to download and add logo
		logoBytes, err := s.downloadLogoFromCDN(context.Background(), *company.LogoURL)
		if err == nil {
			// Create temp file for logo
			tmpFile := fmt.Sprintf("/tmp/logo_%d.png", time.Now().UnixNano())
			pdf.RegisterImageReader(tmpFile, "PNG", bytes.NewReader(logoBytes))
			pdf.Image(tmpFile, 15, 15, 40, 0, false, "", 0, "")
		}
	}

	// Company info on the right
	pdf.SetXY(120, 15)
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(219, 112, 147) // Medium pink
	pdf.Cell(75, 8, s.utf8ToLatin1(company.Name))

	pdf.Ln(6)
	pdf.SetX(120)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(100, 100, 100)

	if company.Email != nil && *company.Email != "" {
		pdf.Cell(75, 5, s.utf8ToLatin1(*company.Email))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.Phone != nil && *company.Phone != "" {
		pdf.Cell(75, 5, s.utf8ToLatin1("Tel: "+*company.Phone))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.WhatsApp != nil && *company.WhatsApp != "" {
		pdf.Cell(75, 5, s.utf8ToLatin1("WhatsApp: "+*company.WhatsApp))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.Instagram != nil && *company.Instagram != "" {
		pdf.Cell(75, 5, s.utf8ToLatin1(*company.Instagram))
	}

	pdf.Ln(15)
	return nil
}

// addTitle adds the budget title
func (s *PDFService) addTitle(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity) {
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(219, 112, 147) // Medium pink
	pdf.Cell(0, 10, s.utf8ToLatin1("ORÇAMENTO"))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 6, s.utf8ToLatin1(budget.Name))
	pdf.Ln(4)

	if budget.Description != "" {
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, s.utf8ToLatin1(budget.Description), "", "", false)
	}

	pdf.Ln(6)
}

// addCustomerInfo adds customer information
func (s *PDFService) addCustomerInfo(pdf *gofpdf.Fpdf, customer *budgetEntities.CustomerInfo) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(219, 112, 147)
	pdf.Cell(0, 8, s.utf8ToLatin1("Cliente"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(60, 60, 60)

	// Name
	pdf.Cell(0, 5, s.utf8ToLatin1(customer.Name))
	pdf.Ln(4)

	// Email
	if customer.Email != nil && *customer.Email != "" {
		pdf.Cell(0, 5, s.utf8ToLatin1("Email: "+*customer.Email))
		pdf.Ln(4)
	}

	// Phone
	if customer.Phone != nil && *customer.Phone != "" {
		pdf.Cell(0, 5, s.utf8ToLatin1("Telefone: "+*customer.Phone))
		pdf.Ln(4)
	}

	// Document
	if customer.Document != nil && *customer.Document != "" {
		pdf.Cell(0, 5, s.utf8ToLatin1("CPF/CNPJ: "+*customer.Document))
	}

	pdf.Ln(8)
}

// addItemsTable adds the items table
func (s *PDFService) addItemsTable(pdf *gofpdf.Fpdf, items []budgetEntities.BudgetItemResponse) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(219, 112, 147)
	pdf.Cell(0, 8, s.utf8ToLatin1("Itens do Orçamento"))
	pdf.Ln(6)

	// Table header
	pdf.SetFillColor(255, 192, 203) // Light pink
	pdf.SetTextColor(255, 255, 255) // White text
	pdf.SetFont("Arial", "B", 9)

	pdf.CellFormat(70, 7, s.utf8ToLatin1("Filamento"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Qtd (g)"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Cor"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Preço/kg"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, s.utf8ToLatin1("Total"), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(60, 60, 60)

	for i, item := range items {
		fillColor := i%2 == 0
		if fillColor {
			pdf.SetFillColor(250, 250, 250) // Very light gray
		}

		filamentName := item.Filament.Name
		if len(filamentName) > 30 {
			filamentName = filamentName[:27] + "..."
		}

		pdf.CellFormat(70, 6, s.utf8ToLatin1(filamentName), "1", 0, "L", fillColor, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.1f", item.Quantity), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(30, 6, s.utf8ToLatin1(item.Filament.Color), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("R$ %.2f", item.Filament.PricePerKg), "1", 0, "R", fillColor, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("R$ %.2f", float64(item.ItemCost)/100.0), "1", 0, "R", fillColor, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(4)
}

// addCostSummary adds the cost summary section
func (s *PDFService) addCostSummary(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(219, 112, 147)
	pdf.Cell(0, 8, s.utf8ToLatin1("Resumo de Custos"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(60, 60, 60)

	// Print time
	printTime := fmt.Sprintf("%dh%02dm", budget.PrintTimeHours, budget.PrintTimeMinutes)
	pdf.Cell(120, 6, s.utf8ToLatin1("Tempo de Impressão:"))
	pdf.Cell(0, 6, printTime)
	pdf.Ln(5)

	// Filament cost
	pdf.Cell(120, 6, s.utf8ToLatin1("Custo de Filamento:"))
	pdf.Cell(0, 6, fmt.Sprintf("R$ %.2f", float64(budget.FilamentCost)/100.0))
	pdf.Ln(5)

	// Waste cost (if included)
	if budget.IncludeWasteCost && budget.WasteCost > 0 {
		pdf.Cell(120, 6, s.utf8ToLatin1("Custo de Desperdício (AMS):"))
		pdf.Cell(0, 6, fmt.Sprintf("R$ %.2f", float64(budget.WasteCost)/100.0))
		pdf.Ln(5)
	}

	// Energy cost (if included)
	if budget.IncludeEnergyCost && budget.EnergyCost > 0 {
		pdf.Cell(120, 6, s.utf8ToLatin1("Custo de Energia:"))
		pdf.Cell(0, 6, fmt.Sprintf("R$ %.2f", float64(budget.EnergyCost)/100.0))
		pdf.Ln(5)
	}

	// Labor cost (if included)
	if budget.IncludeLaborCost && budget.LaborCost > 0 {
		pdf.Cell(120, 6, s.utf8ToLatin1("Custo de Mão de Obra:"))
		pdf.Cell(0, 6, fmt.Sprintf("R$ %.2f", float64(budget.LaborCost)/100.0))
		pdf.Ln(5)
	}

	pdf.Ln(2)

	// Total
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(219, 112, 147)
	pdf.Cell(120, 8, s.utf8ToLatin1("TOTAL:"))
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, fmt.Sprintf("R$ %.2f", float64(budget.TotalCost)/100.0))

	pdf.Ln(8)
}

// addAdditionalInfo adds delivery, payment and notes
func (s *PDFService) addAdditionalInfo(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity) {
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(219, 112, 147)
	pdf.Cell(0, 8, s.utf8ToLatin1("Informações Adicionais"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(60, 60, 60)

	// Delivery days
	if budget.DeliveryDays != nil && *budget.DeliveryDays > 0 {
		pdf.Cell(0, 6, s.utf8ToLatin1(fmt.Sprintf("Prazo de Entrega: %d dias", *budget.DeliveryDays)))
		pdf.Ln(5)
	}

	// Payment terms
	if budget.PaymentTerms != nil && *budget.PaymentTerms != "" {
		pdf.Cell(0, 6, s.utf8ToLatin1("Condições de Pagamento:"))
		pdf.Ln(4)
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, s.utf8ToLatin1(*budget.PaymentTerms), "", "", false)
		pdf.SetFont("Arial", "", 10)
		pdf.Ln(2)
	}

	// Notes
	if budget.Notes != nil && *budget.Notes != "" {
		pdf.Cell(0, 6, s.utf8ToLatin1("Observações:"))
		pdf.Ln(4)
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, s.utf8ToLatin1(*budget.Notes), "", "", false)
	}
}

// addFooter adds footer with company info
func (s *PDFService) addFooter(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo) {
	pdf.SetY(-25)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(150, 150, 150)

	footerText := "Orçamento gerado em " + time.Now().Format("02/01/2006 às 15:04")
	if company.Website != nil && *company.Website != "" {
		footerText += " - " + *company.Website
	}

	pdf.Cell(0, 5, s.utf8ToLatin1(footerText))
	pdf.Ln(4)

	pdf.SetFont("Arial", "", 7)
	pdf.Cell(0, 4, s.utf8ToLatin1("Este orçamento é válido por 15 dias a partir da data de emissão."))
}

// downloadImage downloads an image from URL
func (s *PDFService) downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// downloadLogoFromCDN downloads logo from CDN with authentication
func (s *PDFService) downloadLogoFromCDN(ctx context.Context, logoURL string) ([]byte, error) {
	// Extract path from CDN URL
	// Expected format: https://rb-cdn.rodolfodebonis.com.br/v1/cdn/spooliq/org-{id}/company/logo.{ext}
	// We need: org-{id}/company/logo.{ext}
	
	// Parse URL to extract path after bucket name
	parts := strings.Split(logoURL, "/v1/cdn/spooliq/")
	if len(parts) != 2 {
		s.logger.Error(ctx, "Invalid CDN URL format", map[string]interface{}{
			"url": logoURL,
		})
		return nil, fmt.Errorf("invalid CDN URL format")
	}
	
	path := parts[1]
	
	// Download from CDN with authentication
	logoBytes, err := s.cdnService.DownloadFile(ctx, path)
	if err != nil {
		s.logger.Error(ctx, "Failed to download logo from CDN", map[string]interface{}{
			"error": err.Error(),
			"path":  path,
		})
		return nil, fmt.Errorf("failed to download logo from CDN: %w", err)
	}
	
	return logoBytes, nil
}

// utf8ToLatin1 converts UTF-8 string to Latin1 (required by gofpdf)
func (s *PDFService) utf8ToLatin1(str string) string {
	// Simple conversion - for production, consider using a proper library
	// This handles common Portuguese characters
	replacements := map[rune]string{
		'á': "a", 'à': "a", 'â': "a", 'ã': "a", 'ä': "a",
		'é': "e", 'è': "e", 'ê': "e", 'ë': "e",
		'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
		'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o", 'ö': "o",
		'ú': "u", 'ù': "u", 'û': "u", 'ü': "u",
		'ç': "c",
		'Á': "A", 'À': "A", 'Â': "A", 'Ã': "A", 'Ä': "A",
		'É': "E", 'È': "E", 'Ê': "E", 'Ë': "E",
		'Í': "I", 'Ì': "I", 'Î': "I", 'Ï': "I",
		'Ó': "O", 'Ò': "O", 'Ô': "O", 'Õ': "O", 'Ö': "O",
		'Ú': "U", 'Ù': "U", 'Û': "U", 'Ü': "U",
		'Ç': "C",
	}

	result := ""
	for _, char := range str {
		if replacement, ok := replacements[char]; ok {
			result += replacement
		} else if char < 128 {
			result += string(char)
		}
	}

	return result
}
