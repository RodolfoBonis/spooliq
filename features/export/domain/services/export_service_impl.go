package services

import (
	"context"
	"time"

	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/export/data/services"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/repositories"
	quoteEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	calculationServices "github.com/RodolfoBonis/spooliq/features/calculation/domain/services"
	calculationEntities "github.com/RodolfoBonis/spooliq/features/calculation/domain/entities"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/go-playground/validator/v10"
)

type exportServiceImpl struct {
	quoteRepo          repositories.QuoteRepository
	calculationService calculationServices.CalculationService
	logger             logger.Logger
	validator          *validator.Validate
	jsonService        *services.JSONExportService
	csvService         *services.CSVExportService
	pdfService         *services.PDFExportService
}

// NewExportService cria uma nova instância do serviço de export
func NewExportService(
	quoteRepo repositories.QuoteRepository,
	calculationService calculationServices.CalculationService,
	logger logger.Logger,
) ExportService {
	return &exportServiceImpl{
		quoteRepo:          quoteRepo,
		calculationService: calculationService,
		logger:             logger,
		validator:          validator.New(),
		jsonService:        services.NewJSONExportService(),
		csvService:         services.NewCSVExportService(),
		pdfService:         services.NewPDFExportService(),
	}
}

func (s *exportServiceImpl) ExportQuote(ctx context.Context, request *entities.ExportRequest, userID string) (*entities.ExportResult, error) {
	// Validar request
	if err := s.ValidateRequest(request); err != nil {
		s.logger.LogError(ctx, "Export request validation failed", err)
		return nil, err
	}

	// Buscar quote com dados completos
	quote, err := s.quoteRepo.GetWithFilamentLines(ctx, request.QuoteID, userID)
	if err != nil {
		s.logger.LogError(ctx, "Failed to get quote for export", err)
		return nil, err
	}

	// Preparar dados de export
	exportData := &entities.ExportData{
		Quote: quote,
		Metadata: &entities.ExportMetadata{
			GeneratedAt: time.Now(),
			GeneratedBy: userID,
			Format:      request.Format,
			Version:     "1.0",
			SystemInfo: entities.SystemInfo{
				AppName:    "SpoolIQ",
				AppVersion: "1.0.0",
				Generator:  "Export Service",
			},
		},
	}

	// Incluir cálculos se solicitado
	if request.IncludeCalculation {
		calculationData, err := s.performCalculation(ctx, quote)
		if err != nil {
			// Log do erro mas continua sem os cálculos
			s.logger.Warning(ctx, "Failed to calculate quote for export, continuing without calculations", map[string]interface{}{
				"quote_id": request.QuoteID,
				"error":    err.Error(),
			})
		} else {
			exportData.Calculation = calculationData
		}
	}

	// Gerar export baseado no formato
	var data []byte
	switch request.Format {
	case entities.ExportFormatJSON:
		data, err = s.jsonService.Generate(ctx, exportData)
	case entities.ExportFormatCSV:
		data, err = s.csvService.Generate(ctx, exportData)
	case entities.ExportFormatPDF:
		data, err = s.pdfService.Generate(ctx, exportData)
	default:
		return nil, entities.ErrInvalidFormat
	}

	if err != nil {
		s.logger.LogError(ctx, "Failed to generate export", err)
		return nil, err
	}

	// Criar resultado
	result := &entities.ExportResult{
		Data:        data,
		ContentType: request.Format.GetContentType(),
		Filename:    exportData.GenerateFilename(),
		Size:        int64(len(data)),
		Format:      request.Format,
	}

	s.logger.Info(ctx, "Export generated successfully", map[string]interface{}{
		"quote_id": request.QuoteID,
		"format":   string(request.Format),
		"size":     result.Size,
		"filename": result.Filename,
		"user_id":  userID,
	})

	return result, nil
}

func (s *exportServiceImpl) GetSupportedFormats() []entities.ExportFormat {
	return []entities.ExportFormat{
		entities.ExportFormatJSON,
		entities.ExportFormatCSV,
		entities.ExportFormatPDF,
	}
}

func (s *exportServiceImpl) ValidateRequest(request *entities.ExportRequest) error {
	if request == nil {
		return entities.ErrInvalidExportData
	}

	// Validar usando validator
	if err := s.validator.Struct(request); err != nil {
		return err
	}

	// Validar formato específico
	if !request.Format.IsValid() {
		return entities.ErrInvalidFormat
	}

	return nil
}

// performCalculation executa o cálculo para incluir nos exports
func (s *exportServiceImpl) performCalculation(ctx context.Context, quote *quoteEntities.Quote) (*calculationEntities.CalculationResults, error) {
	// Preparar input de cálculo básico com dados disponíveis
	calculationInput := calculationEntities.CostBreakdown{}

	// Converter filament lines
	for _, line := range quote.FilamentLines {
		filament := calculationEntities.FilamentLineInput{
			Label:      line.FilamentSnapshotName + " " + line.FilamentSnapshotColor,
			Grams:      line.WeightGrams,
			PricePerKg: line.FilamentSnapshotPricePerKg,
		}
		if line.LengthMeters != nil {
			meters := *line.LengthMeters
			filament.Meters = &meters
		}
		if line.FilamentSnapshotPricePerMeter != nil {
			pricePerMeter := *line.FilamentSnapshotPricePerMeter
			filament.PricePerMeter = &pricePerMeter
		}
		calculationInput.Filaments = append(calculationInput.Filaments, filament)
	}

	// Machine profile
	if quote.MachineProfile != nil {
		calculationInput.Machine = calculationEntities.MachineInput{
			Name:         quote.MachineProfile.Name,
			Watt:         quote.MachineProfile.Watt,
			IdleFactor:   quote.MachineProfile.IdleFactor,
			HoursDecimal: 1.0, // Valor padrão para export
		}
	}

	// Energy profile
	if quote.EnergyProfile != nil {
		calculationInput.Energy = calculationEntities.EnergyInput{
			BaseTariff:    quote.EnergyProfile.BaseTariff,
			FlagSurcharge: quote.EnergyProfile.FlagSurcharge,
		}
	}

	// Cost profile
	if quote.CostProfile != nil && quote.MarginProfile != nil {
		calculationInput.Costs = calculationEntities.CostInput{
			WearPct:        quote.CostProfile.WearPercentage,
			Overhead:       quote.CostProfile.OverheadAmount,
			OpRatePerHour:  quote.MarginProfile.OperatorRatePerHour,
			OpMinutes:      0.0, // Valor padrão para export
			CadRatePerHour: quote.MarginProfile.ModelerRatePerHour,
			CadMinutes:     0.0, // Valor padrão para export
		}
	}

	// Margin profile
	if quote.MarginProfile != nil {
		calculationInput.Margins = calculationEntities.MarginInput{
			OnlyPrintPct:     quote.MarginProfile.PrintingOnlyMargin,
			LightAdjustPct:   quote.MarginProfile.PrintingPlusMargin,
			FullModelPct:     quote.MarginProfile.FullServiceMargin,
			ExtraCadLightMin: 30.0, // Valor padrão
			ExtraCadFullMin:  90.0, // Valor padrão
		}
	}

	// Executar cálculo
	result, err := s.calculationService.Calculate(ctx, calculationInput)
	if err != nil {
		return nil, err
	}

	return &result.Results, nil
}