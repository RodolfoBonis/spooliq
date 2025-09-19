package services

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/calculation/domain/entities"
)

// CalculationService implementa todas as fórmulas de cálculo do SpoolIq
type CalculationService interface {
	Calculate(ctx context.Context, input entities.CostBreakdown) (*entities.CostBreakdown, error)
}

type calculationServiceImpl struct {
	logger logger.Logger
}

// NewCalculationService cria uma nova instância do serviço de cálculo
func NewCalculationService(logger logger.Logger) CalculationService {
	return &calculationServiceImpl{
		logger: logger,
	}
}

// Calculate executa o cálculo completo baseado nas fórmulas especificadas
func (s *calculationServiceImpl) Calculate(ctx context.Context, input entities.CostBreakdown) (*entities.CostBreakdown, error) {
	s.logger.Info(ctx, "Iniciando cálculo de custos", map[string]interface{}{
		"filament_lines": len(input.Filaments),
		"machine":        input.Machine.Name,
	})

	result := input

	// 1. Calcular custos de filamentos
	filamentCosts, totalFilamentCost := s.calculateFilamentCosts(input.Filaments)

	// 2. Calcular energia
	kWh := input.Machine.CalculateKWh()
	energyCost := input.Energy.CalculateEnergyCost(kWh)

	// 3. Calcular custo dos materiais (fórmula: Σ CustoFilamento_i + CustoEnergia)
	materialsCost := totalFilamentCost + energyCost

	// 4. Calcular custo de desgaste (fórmula: CustoMateriais * (wear_pct/100))
	wearCost := input.Costs.CalculateWearCost(materialsCost)

	// 5. Calcular custo de mão de obra (fórmula: (op_rate_per_hour * op_minutes/60) + (cad_rate_per_hour * cad_minutes/60))
	laborCost := input.Costs.CalculateLaborCost()

	// 6. Calcular custo direto (fórmula: CustoMateriais + CustoDesgaste + overhead + CustoMaoObra)
	directCost := materialsCost + wearCost + input.Costs.Overhead + laborCost

	// 7. Calcular pacotes de venda (fórmula: CustoDireto * (1 + margem/100))
	packages := s.calculatePackages(directCost, input.Margins, input.Costs.CadRatePerHour)

	// 8. Calcular métricas finais
	markup, effectiveMargin := s.calculateMetrics(directCost, packages)

	// Montar resultado
	result.Results = entities.CalculationResults{
		FilamentCosts:   filamentCosts,
		KWh:             kWh,
		EnergyCost:      energyCost,
		MaterialsCost:   materialsCost,
		WearCost:        wearCost,
		LaborCost:       laborCost,
		DirectCost:      directCost,
		Packages:        packages,
		Markup:          markup,
		EffectiveMargin: effectiveMargin,
	}

	s.logger.Info(ctx, "Cálculo concluído", map[string]interface{}{
		"direct_cost":      directCost,
		"materials_cost":   materialsCost,
		"energy_cost":      energyCost,
		"packages_count":   len(packages),
		"effective_margin": effectiveMargin,
	})

	return &result, nil
}

// calculateFilamentCosts calcula o custo de cada linha de filamento
func (s *calculationServiceImpl) calculateFilamentCosts(filaments []entities.FilamentLineInput) ([]entities.FilamentCostResult, float64) {
	results := make([]entities.FilamentCostResult, len(filaments))
	totalCost := 0.0

	for i, filament := range filaments {
		cost := filament.CalculateCost()
		results[i] = entities.FilamentCostResult{
			Label: filament.Label,
			Cost:  cost,
		}
		totalCost += cost
	}

	return results, totalCost
}

// calculatePackages calcula os preços dos diferentes pacotes de venda
func (s *calculationServiceImpl) calculatePackages(directCost float64, margins entities.MarginInput, cadRatePerHour float64) []entities.PackageResult {
	packages := []entities.PackageResult{
		{
			Type:        "only_print",
			Description: "Somente Impressão",
			Price:       margins.CalculatePackagePrice(directCost, "only_print"),
		},
		{
			Type:        "light_adjust",
			Description: "Impressão + Ajustes Leves",
			Price:       margins.CalculatePackagePrice(directCost, "light_adjust"),
		},
		{
			Type:        "full_model",
			Description: "Impressão + Modelagem Completa",
			Price:       margins.CalculatePackagePrice(directCost, "full_model"),
		},
	}

	// Calcular markup e margem para cada pacote
	for i := range packages {
		packages[i].Markup = entities.GetMarkup(directCost, packages[i].Price)
		packages[i].Margin = entities.GetEffectiveMargin(directCost, packages[i].Price)
	}

	return packages
}

// calculateMetrics calcula as métricas finais baseadas no pacote médio
func (s *calculationServiceImpl) calculateMetrics(directCost float64, packages []entities.PackageResult) (float64, float64) {
	if len(packages) == 0 {
		return 0, 0
	}

	// Usar o pacote de ajustes leves como referência (pacote médio)
	var referencePackage entities.PackageResult
	for _, pkg := range packages {
		if pkg.Type == "light_adjust" {
			referencePackage = pkg
			break
		}
	}

	// Se não encontrou light_adjust, usar o primeiro pacote
	if referencePackage.Type == "" {
		referencePackage = packages[0]
	}

	markup := entities.GetMarkup(directCost, referencePackage.Price)
	effectiveMargin := entities.GetEffectiveMargin(directCost, referencePackage.Price)

	return markup, effectiveMargin
}

// ValidateInput valida os dados de entrada para o cálculo
func (s *calculationServiceImpl) ValidateInput(ctx context.Context, input entities.CostBreakdown) error {
	if len(input.Filaments) == 0 {
		return fmt.Errorf("pelo menos uma linha de filamento é obrigatória")
	}

	for i, filament := range input.Filaments {
		if filament.Grams <= 0 {
			return fmt.Errorf("linha de filamento %d: gramas deve ser maior que zero", i+1)
		}
		if filament.PricePerKg <= 0 {
			return fmt.Errorf("linha de filamento %d: preço por kg deve ser maior que zero", i+1)
		}
		if filament.Meters != nil && *filament.Meters <= 0 {
			return fmt.Errorf("linha de filamento %d: metros deve ser maior que zero quando especificado", i+1)
		}
		if filament.PricePerMeter != nil && *filament.PricePerMeter <= 0 {
			return fmt.Errorf("linha de filamento %d: preço por metro deve ser maior que zero quando especificado", i+1)
		}
	}

	if input.Machine.Watt <= 0 {
		return fmt.Errorf("potência da máquina deve ser maior que zero")
	}

	if input.Machine.HoursDecimal <= 0 {
		return fmt.Errorf("tempo de impressão deve ser maior que zero")
	}

	if input.Energy.BaseTariff <= 0 {
		return fmt.Errorf("tarifa base de energia deve ser maior que zero")
	}

	return nil
}
