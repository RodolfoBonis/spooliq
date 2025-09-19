package services

import (
	"context"
	"testing"

	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/calculation/domain/entities"
)

func TestCalculationService_Calculate(t *testing.T) {
	// Setup
	ctx := context.Background()
	log := logger.NewLogger()
	service := NewCalculationService(log)

	// Caso de teste baseado no exemplo fornecido
	input := entities.CostBreakdown{
		Filaments: []entities.FilamentLineInput{
			{
				Label:      "Cor Principal",
				Grams:      63.53,
				PricePerKg: 125.0,
			},
		},
		Machine: entities.MachineInput{
			Name:         "BambuLab A1 Combo",
			Watt:         95,
			IdleFactor:   0,
			HoursDecimal: 7.233,
		},
		Energy: entities.EnergyInput{
			BaseTariff:    0.804,
			FlagSurcharge: 0,
		},
		Costs: entities.CostInput{
			WearPct:        10,
			Overhead:       8,
			OpRatePerHour:  30,
			OpMinutes:      20,
			CadRatePerHour: 80,
			CadMinutes:     0,
		},
		Margins: entities.MarginInput{
			OnlyPrintPct:     70,
			LightAdjustPct:   90,
			FullModelPct:     120,
			ExtraCadLightMin: 30,
			ExtraCadFullMin:  90,
		},
	}

	// Executar cálculo
	result, err := service.Calculate(ctx, input)

	// Verificar se não houve erro
	if err != nil {
		t.Fatalf("Calculate() erro = %v", err)
	}

	// Verificar resultados esperados
	t.Run("Custo do Filamento", func(t *testing.T) {
		// expectedFilamentCost := (125.0 / 1000) * 63.53 // (price_per_kg / 1000) * grams
		expectedFilamentCost := 7.94125 // valor esperado

		if len(result.Results.FilamentCosts) != 1 {
			t.Errorf("Esperado 1 linha de filamento, obtido %d", len(result.Results.FilamentCosts))
		}

		actualCost := result.Results.FilamentCosts[0].Cost
		if !floatEquals(actualCost, expectedFilamentCost, 0.001) {
			t.Errorf("Custo do filamento = %f, esperado %f", actualCost, expectedFilamentCost)
		}
	})

	t.Run("Consumo de Energia", func(t *testing.T) {
		// expectedKWh := (95.0 / 1000) * 7.233 // (watt / 1000) * hours_decimal
		expectedKWh := 0.687135 // valor esperado

		if !floatEquals(result.Results.KWh, expectedKWh, 0.001) {
			t.Errorf("kWh = %f, esperado %f", result.Results.KWh, expectedKWh)
		}
	})

	t.Run("Custo de Energia", func(t *testing.T) {
		// expectedEnergyCost := 0.687135 * (0.804 + 0) // kWh * (base_tariff + flag_surcharge)
		expectedEnergyCost := 0.5524605 // valor esperado

		if !floatEquals(result.Results.EnergyCost, expectedEnergyCost, 0.001) {
			t.Errorf("Custo de energia = %f, esperado %f", result.Results.EnergyCost, expectedEnergyCost)
		}
	})

	t.Run("Custo dos Materiais", func(t *testing.T) {
		// expectedMaterialsCost := 7.94125 + 0.5524605 // filament_cost + energy_cost
		expectedMaterialsCost := 8.4937105 // valor esperado

		if !floatEquals(result.Results.MaterialsCost, expectedMaterialsCost, 0.001) {
			t.Errorf("Custo dos materiais = %f, esperado %f", result.Results.MaterialsCost, expectedMaterialsCost)
		}
	})

	t.Run("Custo de Desgaste", func(t *testing.T) {
		// expectedWearCost := 8.4937105 * (10.0 / 100) // materials_cost * (wear_pct/100)
		expectedWearCost := 0.84937105 // valor esperado

		if !floatEquals(result.Results.WearCost, expectedWearCost, 0.001) {
			t.Errorf("Custo de desgaste = %f, esperado %f", result.Results.WearCost, expectedWearCost)
		}
	})

	t.Run("Custo de Mão de Obra", func(t *testing.T) {
		// expectedLaborCost := (30.0 * 20.0 / 60) + (80.0 * 0.0 / 60) // (op_rate * op_minutes/60) + (cad_rate * cad_minutes/60)
		expectedLaborCost := 10.0 // valor esperado

		if !floatEquals(result.Results.LaborCost, expectedLaborCost, 0.001) {
			t.Errorf("Custo de mão de obra = %f, esperado %f", result.Results.LaborCost, expectedLaborCost)
		}
	})

	t.Run("Custo Direto", func(t *testing.T) {
		// expectedDirectCost := 8.4937105 + 0.84937105 + 8.0 + 10.0 // materials + wear + overhead + labor
		expectedDirectCost := 27.343082 // valor esperado

		if !floatEquals(result.Results.DirectCost, expectedDirectCost, 0.001) {
			t.Errorf("Custo direto = %f, esperado %f", result.Results.DirectCost, expectedDirectCost)
		}
	})

	t.Run("Pacotes de Venda", func(t *testing.T) {
		if len(result.Results.Packages) != 3 {
			t.Errorf("Esperado 3 pacotes, obtido %d", len(result.Results.Packages))
		}

		// Verificar pacote "só impressão" (70% margem)
		onlyPrintPackage := findPackage(result.Results.Packages, "only_print")
		if onlyPrintPackage == nil {
			t.Error("Pacote 'only_print' não encontrado")
		} else {
			// expectedPrice := 27.343082 * (1 + 70.0/100) // direct_cost * (1 + margin/100)
			expectedPrice := 46.4832394 // valor esperado

			if !floatEquals(onlyPrintPackage.Price, expectedPrice, 0.001) {
				t.Errorf("Preço do pacote 'only_print' = %f, esperado %f", onlyPrintPackage.Price, expectedPrice)
			}
		}

		// Verificar pacote "ajustes leves" (90% margem + 30min CAD)
		lightAdjustPackage := findPackage(result.Results.Packages, "light_adjust")
		if lightAdjustPackage == nil {
			t.Error("Pacote 'light_adjust' não encontrado")
		} else {
			// expectedPrice := 27.343082 * (1 + 90.0/100) // direct_cost * (1 + margin/100)
			// expectedPrice += (80.0 * 30.0) / 60         // + extra CAD time
			// expectedPrice = 51.9518558 + 40.0           // valor esperado
			expectedPrice := 91.9518558

			if !floatEquals(lightAdjustPackage.Price, expectedPrice, 0.001) {
				t.Errorf("Preço do pacote 'light_adjust' = %f, esperado %f", lightAdjustPackage.Price, expectedPrice)
			}
		}
	})
}

func TestCalculationService_MultipleFilaments(t *testing.T) {
	// Setup
	ctx := context.Background()
	log := logger.NewLogger()
	service := NewCalculationService(log)

	// Caso de teste com múltiplos filamentos
	input := entities.CostBreakdown{
		Filaments: []entities.FilamentLineInput{
			{
				Label:      "Cor Principal",
				Grams:      50.0,
				PricePerKg: 125.0,
			},
			{
				Label:      "Cor Secundária",
				Grams:      25.0,
				PricePerKg: 140.0,
			},
			{
				Label:         "Suporte Solúvel",
				Meters:        &[]float64{10.0}[0], // 10 metros
				PricePerMeter: &[]float64{0.25}[0], // R$ 0,25 por metro
				Grams:         30.0,                // será ignorado porque meters está definido
				PricePerKg:    100.0,               // será ignorado porque price_per_meter está definido
			},
		},
		Machine: entities.MachineInput{
			Name:         "Test Machine",
			Watt:         100,
			IdleFactor:   0.1,
			HoursDecimal: 5.0,
		},
		Energy: entities.EnergyInput{
			BaseTariff:    0.8,
			FlagSurcharge: 0.1,
		},
		Costs: entities.CostInput{
			WearPct:        15,
			Overhead:       10,
			OpRatePerHour:  35,
			OpMinutes:      30,
			CadRatePerHour: 80,
			CadMinutes:     15,
		},
		Margins: entities.MarginInput{
			OnlyPrintPct:   80,
			LightAdjustPct: 100,
			FullModelPct:   150,
		},
	}

	// Executar cálculo
	result, err := service.Calculate(ctx, input)

	// Verificar se não houve erro
	if err != nil {
		t.Fatalf("Calculate() erro = %v", err)
	}

	// Verificar custos dos filamentos
	t.Run("Custos Múltiplos Filamentos", func(t *testing.T) {
		if len(result.Results.FilamentCosts) != 3 {
			t.Errorf("Esperado 3 linhas de filamento, obtido %d", len(result.Results.FilamentCosts))
		}

		// Filamento 1: (125 / 1000) * 50 = 6.25
		expectedCost1 := 6.25
		if !floatEquals(result.Results.FilamentCosts[0].Cost, expectedCost1, 0.001) {
			t.Errorf("Custo filamento 1 = %f, esperado %f", result.Results.FilamentCosts[0].Cost, expectedCost1)
		}

		// Filamento 2: (140 / 1000) * 25 = 3.5
		expectedCost2 := 3.5
		if !floatEquals(result.Results.FilamentCosts[1].Cost, expectedCost2, 0.001) {
			t.Errorf("Custo filamento 2 = %f, esperado %f", result.Results.FilamentCosts[1].Cost, expectedCost2)
		}

		// Filamento 3: 0.25 * 10 = 2.5 (usando metros)
		expectedCost3 := 2.5
		if !floatEquals(result.Results.FilamentCosts[2].Cost, expectedCost3, 0.001) {
			t.Errorf("Custo filamento 3 = %f, esperado %f", result.Results.FilamentCosts[2].Cost, expectedCost3)
		}

		// Total de filamentos: 6.25 + 3.5 + 2.5 = 12.25
		expectedTotal := 12.25
		totalFilamentCost := 0.0
		for _, cost := range result.Results.FilamentCosts {
			totalFilamentCost += cost.Cost
		}
		if !floatEquals(totalFilamentCost, expectedTotal, 0.001) {
			t.Errorf("Custo total filamentos = %f, esperado %f", totalFilamentCost, expectedTotal)
		}
	})

	t.Run("Energia com Fator Idle", func(t *testing.T) {
		// kWh = (100 * (1 + 0.1) / 1000) * 5 = (110 / 1000) * 5 = 0.55
		expectedKWh := 0.55
		if !floatEquals(result.Results.KWh, expectedKWh, 0.001) {
			t.Errorf("kWh = %f, esperado %f", result.Results.KWh, expectedKWh)
		}

		// Custo energia = 0.55 * (0.8 + 0.1) = 0.55 * 0.9 = 0.495
		expectedEnergyCost := 0.495
		if !floatEquals(result.Results.EnergyCost, expectedEnergyCost, 0.001) {
			t.Errorf("Custo energia = %f, esperado %f", result.Results.EnergyCost, expectedEnergyCost)
		}
	})
}

func TestCalculationService_ValidateInput(t *testing.T) {
	ctx := context.Background()
	log := logger.NewLogger()
	service := NewCalculationService(log).(*calculationServiceImpl)

	t.Run("Input Válido", func(t *testing.T) {
		input := entities.CostBreakdown{
			Filaments: []entities.FilamentLineInput{
				{
					Label:      "Teste",
					Grams:      10.0,
					PricePerKg: 100.0,
				},
			},
			Machine: entities.MachineInput{
				Watt:         50,
				HoursDecimal: 1.0,
			},
			Energy: entities.EnergyInput{
				BaseTariff: 0.5,
			},
		}

		err := service.ValidateInput(ctx, input)
		if err != nil {
			t.Errorf("ValidateInput() deveria ser válido, erro = %v", err)
		}
	})

	t.Run("Sem Filamentos", func(t *testing.T) {
		input := entities.CostBreakdown{
			Filaments: []entities.FilamentLineInput{},
		}

		err := service.ValidateInput(ctx, input)
		if err == nil {
			t.Error("ValidateInput() deveria retornar erro para entrada sem filamentos")
		}
	})

	t.Run("Gramas Inválidas", func(t *testing.T) {
		input := entities.CostBreakdown{
			Filaments: []entities.FilamentLineInput{
				{
					Label:      "Teste",
					Grams:      0,
					PricePerKg: 100.0,
				},
			},
		}

		err := service.ValidateInput(ctx, input)
		if err == nil {
			t.Error("ValidateInput() deveria retornar erro para gramas inválidas")
		}
	})
}

// Helpers

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < tolerance
}

func findPackage(packages []entities.PackageResult, packageType string) *entities.PackageResult {
	for _, pkg := range packages {
		if pkg.Type == packageType {
			return &pkg
		}
	}
	return nil
}
