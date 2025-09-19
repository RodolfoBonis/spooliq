package entities

// CostBreakdown representa o detalhamento completo de custos de um orçamento
// @Description Detalhamento completo de custos de impressão 3D
type CostBreakdown struct {
	// Entrada de dados
	Filaments []FilamentLineInput `json:"filaments" validate:"required,min=1"`
	Machine   MachineInput        `json:"machine" validate:"required"`
	Energy    EnergyInput         `json:"energy" validate:"required"`
	Costs     CostInput           `json:"costs" validate:"required"`
	Margins   MarginInput         `json:"margins" validate:"required"`

	// Resultados de cálculo
	Results CalculationResults `json:"results"`
}

// FilamentLineInput representa uma linha de filamento para cálculo
// @Description Dados de entrada para linha de filamento
// @Example {"label": "Cor Principal", "grams": 63.53, "meters": null, "price_per_kg": 125.0}
type FilamentLineInput struct {
	Label         string   `json:"label" validate:"required,min=1"`
	Grams         float64  `json:"grams" validate:"required,min=0"`
	Meters        *float64 `json:"meters,omitempty" validate:"omitempty,min=0"`
	PricePerKg    float64  `json:"price_per_kg" validate:"required,min=0"`
	PricePerMeter *float64 `json:"price_per_meter,omitempty" validate:"omitempty,min=0"`
}

// MachineInput representa os dados da impressora para cálculo
// @Description Dados de entrada da impressora
// @Example {"name": "BambuLab A1 Combo", "watt": 95, "idle_factor": 0, "hours_decimal": 7.233}
type MachineInput struct {
	Name         string  `json:"name" validate:"required,min=1"`
	Watt         float64 `json:"watt" validate:"required,min=0"`
	IdleFactor   float64 `json:"idle_factor" validate:"min=0,max=1"`
	HoursDecimal float64 `json:"hours_decimal" validate:"required,min=0"`
}

// EnergyInput representa os dados de energia para cálculo
// @Description Dados de entrada de energia
// @Example {"base_tariff": 0.804, "flag_surcharge": 0}
type EnergyInput struct {
	BaseTariff    float64 `json:"base_tariff" validate:"required,min=0"`
	FlagSurcharge float64 `json:"flag_surcharge" validate:"min=0"`
}

// CostInput representa os custos operacionais para cálculo
// @Description Dados de entrada de custos operacionais
// @Example {"wear_pct": 10, "overhead": 8, "op_rate_per_hour": 30, "op_minutes": 20, "cad_rate_per_hour": 80, "cad_minutes": 0}
type CostInput struct {
	WearPct        float64 `json:"wear_pct" validate:"min=0,max=100"`
	Overhead       float64 `json:"overhead" validate:"min=0"`
	OpRatePerHour  float64 `json:"op_rate_per_hour" validate:"min=0"`
	OpMinutes      float64 `json:"op_minutes" validate:"min=0"`
	CadRatePerHour float64 `json:"cad_rate_per_hour" validate:"min=0"`
	CadMinutes     float64 `json:"cad_minutes" validate:"min=0"`
}

// MarginInput representa as margens para cálculo
// @Description Dados de entrada de margens
// @Example {"only_print_pct": 70, "light_adjust_pct": 90, "full_model_pct": 120, "extra_cad_light_min": 30, "extra_cad_full_min": 90}
type MarginInput struct {
	OnlyPrintPct     float64 `json:"only_print_pct" validate:"min=0"`
	LightAdjustPct   float64 `json:"light_adjust_pct" validate:"min=0"`
	FullModelPct     float64 `json:"full_model_pct" validate:"min=0"`
	ExtraCadLightMin float64 `json:"extra_cad_light_min" validate:"min=0"`
	ExtraCadFullMin  float64 `json:"extra_cad_full_min" validate:"min=0"`
}

// CalculationResults representa os resultados dos cálculos
// @Description Resultados completos dos cálculos
type CalculationResults struct {
	// Custos por linha de filamento
	FilamentCosts []FilamentCostResult `json:"filament_costs"`

	// Energia
	KWh        float64 `json:"kwh"`
	EnergyCost float64 `json:"energy_cost"`

	// Custos agregados
	MaterialsCost float64 `json:"materials_cost"`
	WearCost      float64 `json:"wear_cost"`
	LaborCost     float64 `json:"labor_cost"`
	DirectCost    float64 `json:"direct_cost"`

	// Pacotes de venda
	Packages []PackageResult `json:"packages"`

	// Métricas
	Markup          float64 `json:"markup"`
	EffectiveMargin float64 `json:"effective_margin"`
}

// FilamentCostResult representa o custo calculado de uma linha de filamento
// @Description Resultado do cálculo de custo de filamento
type FilamentCostResult struct {
	Label string  `json:"label"`
	Cost  float64 `json:"cost"`
}

// PackageResult representa um pacote de venda com preço calculado
// @Description Resultado de um pacote de venda
type PackageResult struct {
	Type        string  `json:"type"`        // "only_print", "light_adjust", "full_model"
	Description string  `json:"description"` // Descrição amigável
	Price       float64 `json:"price"`
	Markup      float64 `json:"markup"`
	Margin      float64 `json:"margin"`
}

// CalculateCost calcula o custo de uma linha de filamento baseado nos dados de entrada
func (f *FilamentLineInput) CalculateCost() float64 {
	if f.Meters != nil && f.PricePerMeter != nil && *f.Meters > 0 {
		// Usar preço por metro se disponível
		return *f.PricePerMeter * *f.Meters
	}
	// Usar preço por kg
	return (f.PricePerKg / 1000) * f.Grams
}

// CalculateKWh calcula o consumo de energia em kWh
func (m *MachineInput) CalculateKWh() float64 {
	effectiveWatt := m.Watt * (1 + m.IdleFactor)
	return (effectiveWatt / 1000) * m.HoursDecimal
}

// CalculateEnergyCost calcula o custo da energia
func (e *EnergyInput) CalculateEnergyCost(kWh float64) float64 {
	totalTariff := e.BaseTariff + e.FlagSurcharge
	return kWh * totalTariff
}

// CalculateWearCost calcula o custo de desgaste
func (c *CostInput) CalculateWearCost(materialsCost float64) float64 {
	return materialsCost * (c.WearPct / 100)
}

// CalculateLaborCost calcula o custo de mão de obra
func (c *CostInput) CalculateLaborCost() float64 {
	opCost := (c.OpRatePerHour * c.OpMinutes) / 60
	cadCost := (c.CadRatePerHour * c.CadMinutes) / 60
	return opCost + cadCost
}

// CalculatePackagePrice calcula o preço de um pacote específico
func (m *MarginInput) CalculatePackagePrice(directCost float64, packageType string) float64 {
	var marginPct float64
	var extraCadMinutes float64

	switch packageType {
	case "only_print":
		marginPct = m.OnlyPrintPct
	case "light_adjust":
		marginPct = m.LightAdjustPct
		extraCadMinutes = m.ExtraCadLightMin
	case "full_model":
		marginPct = m.FullModelPct
		extraCadMinutes = m.ExtraCadFullMin
	default:
		marginPct = m.OnlyPrintPct
	}

	// Custo base com margem
	basePriceWithMargin := directCost * (1 + marginPct/100)

	// Adicionar tempo extra de CAD se houver
	if extraCadMinutes > 0 {
		// Assumir taxa de CAD de 80 R$/hora como padrão
		extraCadCost := (80 * extraCadMinutes) / 60
		return basePriceWithMargin + extraCadCost
	}

	return basePriceWithMargin
}

// GetEffectiveMargin calcula a margem efetiva
func GetEffectiveMargin(directCost, finalPrice float64) float64 {
	if directCost <= 0 {
		return 0
	}
	return ((finalPrice - directCost) / directCost) * 100
}

// GetMarkup calcula o markup
func GetMarkup(directCost, finalPrice float64) float64 {
	if finalPrice <= 0 {
		return 0
	}
	return ((finalPrice - directCost) / finalPrice) * 100
}
