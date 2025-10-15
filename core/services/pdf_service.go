package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RodolfoBonis/spooliq/core/logger"
	budgetEntities "github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	companyEntities "github.com/RodolfoBonis/spooliq/features/company/domain/entities"
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
	Branding *companyEntities.CompanyBrandingEntity // NEW: Branding configuration
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
	// Use default branding if not provided
	if data.Branding == nil {
		data.Branding = companyEntities.GetDefaultTemplate()
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// Add header
	if err := s.addHeader(pdf, data.Company, data.Branding); err != nil {
		s.logger.Error(ctx, "Failed to add header to PDF", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Add title
	s.addTitle(pdf, data.Budget, data.Branding)

	// Add customer info
	s.addCustomerInfo(pdf, data.Customer, data.Branding)

	// Add items table
	s.addItemsTable(pdf, data.Items, data.Branding)

	// Add cost summary
	s.addCostSummary(pdf, data.Budget, data.Branding)

	// Add additional info (delivery, payment, notes)
	s.addAdditionalInfo(pdf, data.Budget, data.Branding)

	// Add footer
	s.addFooter(pdf, data.Company, data.Branding)

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
func (s *PDFService) addHeader(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo, branding *companyEntities.CompanyBrandingEntity) error {
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
	r, g, b := s.hexToRGB(branding.PrimaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(75, 8, s.utf8ToLatin1(company.Name))

	pdf.Ln(6)
	pdf.SetX(120)
	pdf.SetFont("Arial", "", 9)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

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
func (s *PDFService) addTitle(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 16)
	r, g, b := s.hexToRGB(branding.TitleColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 10, s.utf8ToLatin1("ORÇAMENTO"))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 6, s.utf8ToLatin1(budget.Name))
	pdf.Ln(4)

	if budget.Description != "" {
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, s.utf8ToLatin1(budget.Description), "", "", false)
	}

	pdf.Ln(6)
}

// addCustomerInfo adds customer information
func (s *PDFService) addCustomerInfo(pdf *gofpdf.Fpdf, customer *budgetEntities.CustomerInfo, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.utf8ToLatin1("Cliente"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

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
func (s *PDFService) addItemsTable(pdf *gofpdf.Fpdf, items []budgetEntities.BudgetItemResponse, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.utf8ToLatin1("Itens do Orçamento"))
	pdf.Ln(6)

	// Table header
	r, g, b = s.hexToRGB(branding.TableHeaderBgColor)
	pdf.SetFillColor(r, g, b)
	r, g, b = s.hexToRGB(branding.HeaderTextColor)
	pdf.SetTextColor(r, g, b)
	pdf.SetFont("Arial", "B", 9)

	pdf.CellFormat(70, 7, s.utf8ToLatin1("Filamento"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Qtd (g)"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Cor"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, s.utf8ToLatin1("Preço/kg"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, s.utf8ToLatin1("Total"), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 8)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	for i, item := range items {
		fillColor := i%2 == 0
		if fillColor {
			r, g, b = s.hexToRGB(branding.TableRowAltBgColor)
			pdf.SetFillColor(r, g, b)
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
func (s *PDFService) addCostSummary(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.utf8ToLatin1("Resumo de Custos"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

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
	r, g, b = s.hexToRGB(branding.AccentColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(120, 8, s.utf8ToLatin1("TOTAL:"))
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, fmt.Sprintf("R$ %.2f", float64(budget.TotalCost)/100.0))

	pdf.Ln(8)
}

// addAdditionalInfo adds delivery, payment and notes
func (s *PDFService) addAdditionalInfo(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.utf8ToLatin1("Informações Adicionais"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

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
func (s *PDFService) addFooter(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetY(-25)
	pdf.SetFont("Arial", "I", 8)
	r, g, b := s.hexToRGB(branding.BorderColor)
	pdf.SetTextColor(r, g, b)

	footerText := "Orçamento gerado em " + time.Now().Format("02/01/2006 às 15:04")
	if company.Website != nil && *company.Website != "" {
		footerText += " - " + *company.Website
	}

	pdf.Cell(0, 5, s.utf8ToLatin1(footerText))
	pdf.Ln(4)

	pdf.SetFont("Arial", "", 7)
	pdf.Cell(0, 4, s.utf8ToLatin1("Este orçamento é válido por 15 dias a partir da data de emissão."))
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

// hexToRGB converts hex color string to RGB values
func (s *PDFService) hexToRGB(hex string) (int, int, int) {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")

	// Default to black if invalid
	if len(hex) != 6 {
		return 0, 0, 0
	}

	// Parse hex to RGB
	var r, g, b int
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
