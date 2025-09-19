package services

import (
	"context"
	"encoding/json"

	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
)

type JSONExportService struct{}

// NewJSONExportService cria uma nova instância do serviço de export JSON
func NewJSONExportService() *JSONExportService {
	return &JSONExportService{}
}

// Generate gera um export em formato JSON
func (s *JSONExportService) Generate(ctx context.Context, data *entities.ExportData) ([]byte, error) {
	if err := data.Validate(); err != nil {
		return nil, err
	}

	// Estrutura otimizada para JSON export
	exportJSON := map[string]interface{}{
		"metadata": data.Metadata,
		"quote": map[string]interface{}{
			"id":              data.Quote.ID,
			"title":           data.Quote.Title,
			"notes":           data.Quote.Notes,
			"owner_user_id":   data.Quote.OwnerUserID,
			"created_at":      data.Quote.CreatedAt,
			"updated_at":      data.Quote.UpdatedAt,
			"filament_lines":  data.Quote.FilamentLines,
			"machine_profile": data.Quote.MachineProfile,
			"energy_profile":  data.Quote.EnergyProfile,
			"cost_profile":    data.Quote.CostProfile,
			"margin_profile":  data.Quote.MarginProfile,
		},
	}

	// Adicionar cálculos se disponíveis
	if data.Calculation != nil {
		exportJSON["calculation"] = data.Calculation
	}

	// Serializar para JSON com formatação bonita
	jsonData, err := json.MarshalIndent(exportJSON, "", "  ")
	if err != nil {
		return nil, entities.ErrExportGeneration
	}

	return jsonData, nil
}

// GetContentType retorna o content type para JSON
func (s *JSONExportService) GetContentType() string {
	return "application/json"
}

// GetFileExtension retorna a extensão para arquivos JSON
func (s *JSONExportService) GetFileExtension() string {
	return "json"
}
