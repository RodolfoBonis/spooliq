package services

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/jung-kurt/gofpdf"
	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
)

type PDFExportService struct{}

// NewPDFExportService cria uma nova instância do serviço de export PDF
func NewPDFExportService() *PDFExportService {
	return &PDFExportService{}
}

// Generate gera um export em formato PDF
func (s *PDFExportService) Generate(ctx context.Context, data *entities.ExportData) ([]byte, error) {
	if err := data.Validate(); err != nil {
		return nil, err
	}

	// Criar novo documento PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Adicionar header
	s.addHeader(pdf, data)

	// Adicionar informações da quote
	s.addQuoteInfo(pdf, data)

	// Adicionar filamentos
	s.addFilamentLines(pdf, data)

	// Adicionar perfis
	s.addProfiles(pdf, data)

	// Adicionar cálculos se disponíveis
	if data.Calculation != nil {
		s.addCalculationResults(pdf, data)
	}

	// Adicionar footer
	s.addFooter(pdf, data)

	// Gerar o PDF em buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, entities.ErrExportGeneration
	}

	return buf.Bytes(), nil
}

func (s *PDFExportService) addHeader(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	// Título principal
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(0, 100, 150)
	pdf.Cell(0, 15, "SpoolIQ - Orçamento de Impressão 3D")
	pdf.Ln(20)

	// Informações do documento
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(50, 5, "Gerado em: "+data.Metadata.GeneratedAt.Format("02/01/2006 15:04:05"))
	pdf.Ln(5)
	pdf.Cell(50, 5, "Por: "+data.Metadata.GeneratedBy)
	pdf.Ln(10)

	// Linha separadora
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(10)
}

func (s *PDFExportService) addQuoteInfo(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	// Seção de informações da quote
	pdf.SetFont("Arial", "B", 14)
	pdf.SetTextColor(0, 0, 0)
	pdf.Cell(0, 8, "Informações do Orçamento")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 10)

	// ID e Título
	pdf.Cell(30, 6, "ID:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, strconv.FormatUint(uint64(data.Quote.ID), 10))
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(30, 6, "Título:")
	pdf.SetFont("Arial", "B", 10)
	pdf.Cell(0, 6, data.Quote.Title)
	pdf.Ln(8)

	// Datas
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(30, 6, "Criado em:")
	pdf.Cell(50, 6, data.Quote.CreatedAt.Format("02/01/2006 15:04:05"))
	pdf.Cell(30, 6, "Atualizado em:")
	pdf.Cell(0, 6, data.Quote.UpdatedAt.Format("02/01/2006 15:04:05"))
	pdf.Ln(8)

	// Observações se houver
	if data.Quote.Notes != "" {
		pdf.Cell(30, 6, "Observações:")
		pdf.Ln(6)
		pdf.SetFont("Arial", "I", 9)
		pdf.MultiCell(0, 5, data.Quote.Notes, "0", "L", false)
		pdf.Ln(5)
	}

	pdf.Ln(5)
}

func (s *PDFExportService) addFilamentLines(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	if len(data.Quote.FilamentLines) == 0 {
		return
	}

	// Título da seção
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Filamentos")
	pdf.Ln(12)

	// Cabeçalho da tabela
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)

	colWidths := []float64{50, 25, 20, 25, 25, 25}
	headers := []string{"Nome", "Marca", "Material", "Cor", "Peso (g)", "Preço/Kg"}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(8)

	// Dados dos filamentos
	pdf.SetFont("Arial", "", 8)
	pdf.SetFillColor(255, 255, 255)

	for _, line := range data.Quote.FilamentLines {
		values := []string{
			s.truncateString(line.FilamentSnapshotName, 25),
			s.truncateString(line.FilamentSnapshotBrand, 15),
			s.truncateString(line.FilamentSnapshotMaterial, 10),
			s.truncateString(line.FilamentSnapshotColor, 15),
			fmt.Sprintf("%.1f", line.WeightGrams),
			fmt.Sprintf("R$ %.2f", line.FilamentSnapshotPricePerKg),
		}

		for i, value := range values {
			pdf.CellFormat(colWidths[i], 6, value, "1", 0, "C", false, 0, "")
		}
		pdf.Ln(6)
	}

	pdf.Ln(8)
}

func (s *PDFExportService) addProfiles(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	// Perfil da máquina
	if data.Quote.MachineProfile != nil {
		s.addProfileSection(pdf, "Perfil da Máquina", [][]string{
			{"Nome", data.Quote.MachineProfile.Name},
			{"Marca", data.Quote.MachineProfile.Brand},
			{"Modelo", data.Quote.MachineProfile.Model},
			{"Potência", fmt.Sprintf("%.0f W", data.Quote.MachineProfile.Watt)},
			{"Fator Ocioso", fmt.Sprintf("%.2f", data.Quote.MachineProfile.IdleFactor)},
		})
	}

	// Perfil de energia
	if data.Quote.EnergyProfile != nil {
		s.addProfileSection(pdf, "Perfil de Energia", [][]string{
			{"Tarifa Base", fmt.Sprintf("R$ %.3f/kWh", data.Quote.EnergyProfile.BaseTariff)},
			{"Taxa Bandeira", fmt.Sprintf("R$ %.3f/kWh", data.Quote.EnergyProfile.FlagSurcharge)},
			{"Localização", data.Quote.EnergyProfile.Location},
			{"Ano", strconv.Itoa(data.Quote.EnergyProfile.Year)},
		})
	}

	// Perfil de custos
	if data.Quote.CostProfile != nil {
		s.addProfileSection(pdf, "Perfil de Custos", [][]string{
			{"Desgaste", fmt.Sprintf("%.2f%%", data.Quote.CostProfile.WearPercentage)},
			{"Custos Fixos", fmt.Sprintf("R$ %.2f", data.Quote.CostProfile.OverheadAmount)},
		})
	}

	// Perfil de margens
	if data.Quote.MarginProfile != nil {
		s.addProfileSection(pdf, "Perfil de Margens", [][]string{
			{"Só Impressão", fmt.Sprintf("%.2f%%", data.Quote.MarginProfile.PrintingOnlyMargin)},
			{"Impressão Plus", fmt.Sprintf("%.2f%%", data.Quote.MarginProfile.PrintingPlusMargin)},
			{"Serviço Completo", fmt.Sprintf("%.2f%%", data.Quote.MarginProfile.FullServiceMargin)},
			{"Taxa Operador", fmt.Sprintf("R$ %.2f/h", data.Quote.MarginProfile.OperatorRatePerHour)},
			{"Taxa Modelador", fmt.Sprintf("R$ %.2f/h", data.Quote.MarginProfile.ModelerRatePerHour)},
		})
	}
}

func (s *PDFExportService) addProfileSection(pdf *gofpdf.Fpdf, title string, data [][]string) {
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, title)
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 9)
	for _, row := range data {
		pdf.Cell(50, 6, row[0]+":")
		pdf.Cell(0, 6, row[1])
		pdf.Ln(6)
	}
	pdf.Ln(5)
}

func (s *PDFExportService) addCalculationResults(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	calc := data.Calculation

	// Verificar se há espaço suficiente na página
	if pdf.GetY() > 200 {
		pdf.AddPage()
	}

	// Título da seção
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, "Resultados dos Cálculos")
	pdf.Ln(12)

	// Custos principais
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(240, 240, 240)

	costs := [][]string{
		{"Custo de Materiais", fmt.Sprintf("R$ %.2f", calc.MaterialsCost)},
		{"Custo de Energia", fmt.Sprintf("R$ %.2f", calc.EnergyCost)},
		{"Custo de Desgaste", fmt.Sprintf("R$ %.2f", calc.WearCost)},
		{"Custo de Mão de Obra", fmt.Sprintf("R$ %.2f", calc.LaborCost)},
	}

	for _, cost := range costs {
		pdf.CellFormat(80, 8, cost[0], "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 8, cost[1], "1", 1, "R", true, 0, "")
	}

	// Custo direto total
	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(200, 220, 255)
	pdf.CellFormat(80, 10, "CUSTO DIRETO TOTAL", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 10, fmt.Sprintf("R$ %.2f", calc.DirectCost), "1", 1, "R", true, 0, "")

	pdf.Ln(8)

	// Pacotes se disponíveis
	if len(calc.Packages) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 8, "Pacotes de Preços")
		pdf.Ln(10)

		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(230, 230, 230)

		// Cabeçalho
		pdf.CellFormat(50, 8, "Tipo de Serviço", "1", 0, "C", true, 0, "")
		pdf.CellFormat(40, 8, "Preço", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 8, "Margem", "1", 1, "C", true, 0, "")

		// Dados dos pacotes
		pdf.SetFont("Arial", "", 9)
		pdf.SetFillColor(255, 255, 255)

		for _, pkg := range calc.Packages {
			pdf.CellFormat(50, 6, pkg.Description, "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 6, fmt.Sprintf("R$ %.2f", pkg.Price), "1", 0, "R", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("%.1f%%", pkg.Margin), "1", 1, "R", false, 0, "")
		}
	}
}

func (s *PDFExportService) addFooter(pdf *gofpdf.Fpdf, data *entities.ExportData) {
	// Posicionar no final da página
	pdf.SetY(-20)

	// Linha separadora
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// Informações do sistema
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(100, 100, 100)
	pdf.Cell(0, 5, fmt.Sprintf("Gerado por %s %s - %s",
		data.Metadata.SystemInfo.AppName,
		data.Metadata.SystemInfo.AppVersion,
		data.Metadata.SystemInfo.Generator))
}

func (s *PDFExportService) truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen-3] + "..."
}

// GetContentType retorna o content type para PDF
func (s *PDFExportService) GetContentType() string {
	return "application/pdf"
}

// GetFileExtension retorna a extensão para arquivos PDF
func (s *PDFExportService) GetFileExtension() string {
	return "pdf"
}