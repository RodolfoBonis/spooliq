package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
)

// CSVExportService provides CSV export functionality
type CSVExportService struct{}

// NewCSVExportService cria uma nova instância do serviço de export CSV
func NewCSVExportService() *CSVExportService {
	return &CSVExportService{}
}

// Generate gera um export em formato CSV
func (s *CSVExportService) Generate(ctx context.Context, data *entities.ExportData) ([]byte, error) {
	if err := data.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Escrever cabeçalho de metadados
	if err := s.writeMetadata(writer, data); err != nil {
		return nil, err
	}

	// Escrever dados da quote
	if err := s.writeQuoteData(writer, data); err != nil {
		return nil, err
	}

	// Escrever linhas de filamento
	if err := s.writeFilamentLines(writer, data); err != nil {
		return nil, err
	}

	// Escrever perfis
	if err := s.writeProfiles(writer, data); err != nil {
		return nil, err
	}

	// Escrever cálculos se disponíveis
	if data.Calculation != nil {
		if err := s.writeCalculationData(writer, data); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, entities.ErrExportGeneration
	}

	return buf.Bytes(), nil
}

func (s *CSVExportService) writeMetadata(writer *csv.Writer, data *entities.ExportData) error {
	// Seção de metadados
	records := [][]string{
		{"=== METADADOS ==="},
		{"Data de Geração", data.Metadata.GeneratedAt.Format("02/01/2006 15:04:05")},
		{"Gerado por", data.Metadata.GeneratedBy},
		{"Formato", string(data.Metadata.Format)},
		{"Versão", data.Metadata.Version},
		{"Sistema", data.Metadata.SystemInfo.AppName + " " + data.Metadata.SystemInfo.AppVersion},
		{""},
	}

	return writer.WriteAll(records)
}

func (s *CSVExportService) writeQuoteData(writer *csv.Writer, data *entities.ExportData) error {
	// Seção da quote
	records := [][]string{
		{"=== ORÇAMENTO ==="},
		{"ID", strconv.FormatUint(uint64(data.Quote.ID), 10)},
		{"Título", data.Quote.Title},
		{"Observações", data.Quote.Notes},
		{"Proprietário", data.Quote.OwnerUserID},
		{"Criado em", data.Quote.CreatedAt.Format("02/01/2006 15:04:05")},
		{"Atualizado em", data.Quote.UpdatedAt.Format("02/01/2006 15:04:05")},
		{""},
	}

	return writer.WriteAll(records)
}

func (s *CSVExportService) writeFilamentLines(writer *csv.Writer, data *entities.ExportData) error {
	if len(data.Quote.FilamentLines) == 0 {
		return nil
	}

	// Seção de filamentos
	records := [][]string{
		{"=== FILAMENTOS ==="},
		{"Nome", "Marca", "Material", "Cor", "Cor Hex", "Preço/Kg", "Preço/Metro", "Peso (g)", "Comprimento (m)"},
	}

	for _, line := range data.Quote.FilamentLines {
		pricePerMeter := ""
		if line.FilamentSnapshotPricePerMeter != nil {
			pricePerMeter = fmt.Sprintf("%.4f", *line.FilamentSnapshotPricePerMeter)
		}

		lengthMeters := ""
		if line.LengthMeters != nil {
			lengthMeters = fmt.Sprintf("%.3f", *line.LengthMeters)
		}

		record := []string{
			line.FilamentSnapshotName,
			line.FilamentSnapshotBrand,
			line.FilamentSnapshotMaterial,
			line.FilamentSnapshotColor,
			line.FilamentSnapshotColorHex,
			fmt.Sprintf("%.2f", line.FilamentSnapshotPricePerKg),
			pricePerMeter,
			fmt.Sprintf("%.3f", line.WeightGrams),
			lengthMeters,
		}
		records = append(records, record)
	}

	records = append(records, []string{""})
	return writer.WriteAll(records)
}

func (s *CSVExportService) writeProfiles(writer *csv.Writer, data *entities.ExportData) error {
	var records [][]string

	// Perfil da máquina
	if data.Quote.MachineProfile != nil {
		mp := data.Quote.MachineProfile
		records = append(records, [][]string{
			{"=== PERFIL DA MÁQUINA ==="},
			{"Nome", mp.Name},
			{"Marca", mp.Brand},
			{"Modelo", mp.Model},
			{"Potência (W)", fmt.Sprintf("%.0f", mp.Watt)},
			{"Fator Ocioso", fmt.Sprintf("%.2f", mp.IdleFactor)},
			{"Descrição", mp.Description},
			{"URL", mp.URL},
			{""},
		}...)
	}

	// Perfil de energia
	if data.Quote.EnergyProfile != nil {
		ep := data.Quote.EnergyProfile
		records = append(records, [][]string{
			{"=== PERFIL DE ENERGIA ==="},
			{"Tarifa Base", fmt.Sprintf("%.3f", ep.BaseTariff)},
			{"Taxa Bandeira", fmt.Sprintf("%.3f", ep.FlagSurcharge)},
			{"Localização", ep.Location},
			{"Ano", strconv.Itoa(ep.Year)},
			{"Descrição", ep.Description},
			{""},
		}...)
	}

	// Perfil de custos
	if data.Quote.CostProfile != nil {
		cp := data.Quote.CostProfile
		records = append(records, [][]string{
			{"=== PERFIL DE CUSTOS ==="},
			{"Desgaste (%)", fmt.Sprintf("%.2f", cp.WearPercentage)},
			{"Custos Fixos", fmt.Sprintf("%.2f", cp.OverheadAmount)},
			{"Descrição", cp.Description},
			{""},
		}...)
	}

	// Perfil de margens
	if data.Quote.MarginProfile != nil {
		mp := data.Quote.MarginProfile
		records = append(records, [][]string{
			{"=== PERFIL DE MARGENS ==="},
			{"Margem Só Impressão (%)", fmt.Sprintf("%.2f", mp.PrintingOnlyMargin)},
			{"Margem Impressão Plus (%)", fmt.Sprintf("%.2f", mp.PrintingPlusMargin)},
			{"Margem Serviço Completo (%)", fmt.Sprintf("%.2f", mp.FullServiceMargin)},
			{"Taxa Operador (R$/h)", fmt.Sprintf("%.2f", mp.OperatorRatePerHour)},
			{"Taxa Modelador (R$/h)", fmt.Sprintf("%.2f", mp.ModelerRatePerHour)},
			{"Descrição", mp.Description},
			{""},
		}...)
	}

	if len(records) > 0 {
		return writer.WriteAll(records)
	}

	return nil
}

func (s *CSVExportService) writeCalculationData(writer *csv.Writer, data *entities.ExportData) error {
	calc := data.Calculation

	records := [][]string{
		{"=== RESULTADOS DOS CÁLCULOS ==="},
		{"Custo de Materiais", fmt.Sprintf("R$ %.2f", calc.MaterialsCost)},
		{"Custo de Energia", fmt.Sprintf("R$ %.2f", calc.EnergyCost)},
		{"Custo de Desgaste", fmt.Sprintf("R$ %.2f", calc.WearCost)},
		{"Custo de Mão de Obra", fmt.Sprintf("R$ %.2f", calc.LaborCost)},
		{"Custo Direto Total", fmt.Sprintf("R$ %.2f", calc.DirectCost)},
		{"Energia Consumida", fmt.Sprintf("%.3f kWh", calc.KWh)},
		{"Markup", fmt.Sprintf("%.2f%%", calc.Markup)},
		{"Margem Efetiva", fmt.Sprintf("%.2f%%", calc.EffectiveMargin)},
		{""},
	}

	// Custos detalhados por filamento
	if len(calc.FilamentCosts) > 0 {
		records = append(records, []string{"=== CUSTOS POR FILAMENTO ==="})
		records = append(records, []string{"Filamento", "Custo"})

		for _, fc := range calc.FilamentCosts {
			records = append(records, []string{
				fc.Label,
				fmt.Sprintf("R$ %.2f", fc.Cost),
			})
		}
		records = append(records, []string{""})
	}

	// Pacotes de preços
	if len(calc.Packages) > 0 {
		records = append(records, []string{"=== PACOTES DE PREÇOS ==="})
		records = append(records, []string{"Tipo", "Descrição", "Preço", "Markup", "Margem"})

		for _, pkg := range calc.Packages {
			records = append(records, []string{
				pkg.Type,
				pkg.Description,
				fmt.Sprintf("R$ %.2f", pkg.Price),
				fmt.Sprintf("%.2f%%", pkg.Markup),
				fmt.Sprintf("%.2f%%", pkg.Margin),
			})
		}
	}

	return writer.WriteAll(records)
}

// GetContentType retorna o content type para CSV
func (s *CSVExportService) GetContentType() string {
	return "text/csv"
}

// GetFileExtension retorna a extensão para arquivos CSV
func (s *CSVExportService) GetFileExtension() string {
	return "csv"
}
