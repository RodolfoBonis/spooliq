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
	translator func(string) string // UTF-8 translator for special characters
}

// BudgetPDFData contains all data needed to generate a budget PDF
type BudgetPDFData struct {
	Budget                *budgetEntities.BudgetEntity
	Customer              *budgetEntities.CustomerInfo
	Items                 []budgetEntities.BudgetItemResponse
	Company               *budgetEntities.CompanyInfo
	Branding              *companyEntities.CompanyBrandingEntity // Branding configuration
	TotalPrintTimeHours   int                                    // Total print time (sum of all items)
	TotalPrintTimeMinutes int                                    // Total print time (sum of all items)
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

	// Create PDF with UTF-8 support
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Configure UTF-8 translator for special characters
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(false, 0) // Disable automatic page breaks
	pdf.AddPage()

	// Store translator for UTF-8 conversion
	s.translator = tr

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

	// Add footer at the bottom of the page
	s.addFooterAtBottom(pdf, data.Company, data.Branding)

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
	pdf.Cell(75, 8, s.convertUTF8(company.Name))

	pdf.Ln(6)
	pdf.SetX(120)
	pdf.SetFont("Arial", "", 9)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	if company.Email != nil && *company.Email != "" {
		pdf.Cell(75, 5, s.convertUTF8(*company.Email))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.Phone != nil && *company.Phone != "" {
		pdf.Cell(75, 5, s.convertUTF8("Tel: "+*company.Phone))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.WhatsApp != nil && *company.WhatsApp != "" {
		pdf.Cell(75, 5, s.convertUTF8("WhatsApp: "+*company.WhatsApp))
		pdf.Ln(4)
		pdf.SetX(120)
	}

	if company.Instagram != nil && *company.Instagram != "" {
		pdf.Cell(75, 5, s.convertUTF8(*company.Instagram))
	}

	pdf.Ln(15)
	return nil
}

// addTitle adds the budget title
func (s *PDFService) addTitle(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 16)
	r, g, b := s.hexToRGB(branding.TitleColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 10, s.convertUTF8("ORÇAMENTO"))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 6, s.convertUTF8(budget.Name))
	pdf.Ln(4)

	if budget.Description != "" {
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, s.convertUTF8(budget.Description), "", "", false)
	}

	pdf.Ln(6)
}

// addCustomerInfo adds customer information
func (s *PDFService) addCustomerInfo(pdf *gofpdf.Fpdf, customer *budgetEntities.CustomerInfo, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.convertUTF8("Cliente"))
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 10)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	// Name
	pdf.Cell(0, 5, s.convertUTF8(customer.Name))
	pdf.Ln(4)

	// Email
	if customer.Email != nil && *customer.Email != "" {
		pdf.Cell(0, 5, s.convertUTF8("Email: "+*customer.Email))
		pdf.Ln(4)
	}

	// Phone
	if customer.Phone != nil && *customer.Phone != "" {
		pdf.Cell(0, 5, s.convertUTF8("Telefone: "+*customer.Phone))
		pdf.Ln(4)
	}

	// Document
	if customer.Document != nil && *customer.Document != "" {
		pdf.Cell(0, 5, s.convertUTF8("CPF/CNPJ: "+*customer.Document))
	}

	pdf.Ln(8)
}

// addItemsTable adds the items table showing products (customer-facing)
func (s *PDFService) addItemsTable(pdf *gofpdf.Fpdf, items []budgetEntities.BudgetItemResponse, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 10) // Reduced from 11
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 6, s.convertUTF8("Itens do Orçamento")) // Reduced height from 8 to 6
	pdf.Ln(8)                                           // Reduced from 6 to 4

	// Table header - showing products, not filaments
	r, g, b = s.hexToRGB(branding.TableHeaderBgColor)
	pdf.SetFillColor(r, g, b)
	r, g, b = s.hexToRGB(branding.HeaderTextColor)
	pdf.SetTextColor(r, g, b)
	pdf.SetFont("Arial", "B", 8) // Reduced from 9

	pdf.CellFormat(95, 6, s.convertUTF8("Descrição"), "1", 0, "C", true, 0, "") // Reduced height from 7 to 6
	pdf.CellFormat(20, 6, s.convertUTF8("Qtd."), "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 6, s.convertUTF8("Valor Unitário (R$)"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 6, s.convertUTF8("Subtotal (R$)"), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Table rows - showing products
	pdf.SetFont("Arial", "", 7) // Reduced from 8
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	for i, item := range items {
		fillColor := i%2 == 0
		if fillColor {
			r, g, b = s.hexToRGB(branding.TableRowAltBgColor)
			pdf.SetFillColor(r, g, b)
		}

		// Build product description
		description := item.ProductName
		if item.ProductDimensions != nil && *item.ProductDimensions != "" {
			description += " - " + *item.ProductDimensions
		}

		// Truncate if too long
		if len(description) > 60 {
			description = description[:57] + "..."
		}

		// Calculate subtotal (ProductQuantity * UnitPrice)
		subtotal := float64(item.ProductQuantity) * (float64(item.UnitPrice) / 100.0)

		pdf.CellFormat(95, 5, s.convertUTF8(description), "1", 0, "L", fillColor, 0, "") // Reduced height from 6 to 5
		pdf.CellFormat(20, 5, fmt.Sprintf("%d", item.ProductQuantity), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(35, 5, fmt.Sprintf("%.2f", float64(item.UnitPrice)/100.0), "1", 0, "R", fillColor, 0, "")
		pdf.CellFormat(35, 5, fmt.Sprintf("%.2f", subtotal), "1", 0, "R", fillColor, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(3) // Reduced from 4 to 3
}

// addCostSummary adds the cost summary section
func (s *PDFService) addCostSummary(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.Ln(2)

	// Total
	pdf.SetFont("Arial", "B", 11) // Reduced from 12
	r, g, b := s.hexToRGB(branding.AccentColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(120, 6, s.convertUTF8("TOTAL:")) // Reduced height from 8 to 6
	pdf.SetFont("Arial", "B", 13)             // Reduced from 14
	pdf.Cell(0, 6, fmt.Sprintf("R$ %.2f", float64(budget.TotalCost)/100.0))

	pdf.Ln(6) // Reduced from 8 to 6
}

// addAdditionalInfo adds delivery, payment and notes
func (s *PDFService) addAdditionalInfo(pdf *gofpdf.Fpdf, budget *budgetEntities.BudgetEntity, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 10) // Reduced from 11
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 6, s.convertUTF8("Informações Adicionais")) // Reduced height from 8 to 6
	pdf.Ln(4)                                               // Reduced from 6 to 4

	pdf.SetFont("Arial", "", 9) // Reduced from 10
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	// Delivery days
	if budget.DeliveryDays != nil && *budget.DeliveryDays > 0 {
		pdf.Cell(0, 5, s.convertUTF8(fmt.Sprintf("Prazo de Entrega: %d dias", *budget.DeliveryDays))) // Reduced height from 6 to 5
		pdf.Ln(4)                                                                                     // Reduced from 5 to 4
	}

	// Payment terms
	if budget.PaymentTerms != nil && *budget.PaymentTerms != "" {
		pdf.Cell(0, 5, s.convertUTF8("Condições de Pagamento:"))                // Reduced height from 6 to 5
		pdf.Ln(3)                                                               // Reduced from 4 to 3
		pdf.SetFont("Arial", "I", 8)                                            // Reduced from 9
		pdf.MultiCell(0, 4, s.convertUTF8(*budget.PaymentTerms), "", "", false) // Reduced height from 5 to 4
		pdf.SetFont("Arial", "", 9)
		pdf.Ln(1) // Reduced from 2 to 1
	}

	// Notes
	if budget.Notes != nil && *budget.Notes != "" {
		pdf.Cell(0, 5, s.convertUTF8("Observações:"))                    // Reduced height from 6 to 5
		pdf.Ln(3)                                                        // Reduced from 4 to 3
		pdf.SetFont("Arial", "I", 8)                                     // Reduced from 9
		pdf.MultiCell(0, 4, s.convertUTF8(*budget.Notes), "", "", false) // Reduced height from 5 to 4
	}
}

// addFooter adds footer with company info (deprecated - use addFooterAtBottom)
func (s *PDFService) addFooter(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetY(-20) // Reduzido de -25 para -20
	pdf.SetFont("Arial", "I", 8)
	r, g, b := s.hexToRGB(branding.BorderColor)
	pdf.SetTextColor(r, g, b)

	footerText := "Orçamento gerado em " + time.Now().Format("02/01/2006 às 15:04")
	if company.Website != nil && *company.Website != "" {
		footerText += " - " + *company.Website
	}

	pdf.Cell(0, 4, s.convertUTF8(footerText))
	pdf.Ln(2) // Reduzido de 3 para 2

	pdf.SetFont("Arial", "", 7)
	pdf.Cell(0, 3, s.convertUTF8("Este orçamento é válido por 15 dias a partir da data de emissão."))
}

// addFooterAtBottom adds footer always at the bottom of the page
func (s *PDFService) addFooterAtBottom(pdf *gofpdf.Fpdf, company *budgetEntities.CompanyInfo, branding *companyEntities.CompanyBrandingEntity) {
	// A4 page height is 297mm, with 15mm margin = 267mm usable height
	// Position footer at Y = 270mm to ensure it's at the bottom
	pdf.SetY(270)

	pdf.SetFont("Arial", "I", 8)
	r, g, b := s.hexToRGB(branding.BorderColor)
	pdf.SetTextColor(r, g, b)

	footerText := "Orçamento gerado em " + time.Now().Format("02/01/2006 às 15:04")
	if company.Website != nil && *company.Website != "" {
		footerText += " - " + *company.Website
	}

	pdf.Cell(0, 4, s.convertUTF8(footerText))
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 7)
	pdf.Cell(0, 3, s.convertUTF8("Este orçamento é válido por 15 dias a partir da data de emissão."))
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

// convertUTF8 converts UTF-8 strings for PDF display
func (s *PDFService) convertUTF8(str string) string {
	if s.translator != nil {
		return s.translator(str)
	}
	return str
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
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		// Return black color as default on parsing error
		return 0, 0, 0
	}
	return r, g, b
}
