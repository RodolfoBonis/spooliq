package entities

import (
	"errors"

	"github.com/google/uuid"
)

// CostPresetEntity represents fixed operational costs for 3D printing
type CostPresetEntity struct {
	ID                        uuid.UUID `json:"id"`
	OrganizationID            string    `json:"organization_id"` // Multi-tenancy
	LaborCostPerHour          float32   `json:"labor_cost_per_hour"`
	PackagingCostPerItem      float32   `json:"packaging_cost_per_item"`
	ShippingCostBase          float32   `json:"shipping_cost_base"`
	ShippingCostPerGram       float32   `json:"shipping_cost_per_gram"`
	OverheadPercentage        float32   `json:"overhead_percentage"`
	ProfitMarginPercentage    float32   `json:"profit_margin_percentage"`
	PostProcessingCostPerHour float32   `json:"post_processing_cost_per_hour"`
	SupportRemovalCostPerHour float32   `json:"support_removal_cost_per_hour"`
	QualityControlCostPerItem float32   `json:"quality_control_cost_per_item"`
	FailureRatePercentage     float32   `json:"failure_rate_percentage"`
	WasteGramsPerColorChange  float32   `json:"waste_grams_per_color_change"`
}

// Validate validates the cost preset entity
func (c *CostPresetEntity) Validate() error {
	if c.LaborCostPerHour < 0 {
		return errors.New("o custo de mão de obra por hora não pode ser negativo")
	}
	if c.PackagingCostPerItem < 0 {
		return errors.New("o custo de embalagem por item não pode ser negativo")
	}
	if c.ShippingCostBase < 0 {
		return errors.New("o custo base de envio não pode ser negativo")
	}
	if c.ShippingCostPerGram < 0 {
		return errors.New("o custo de envio por grama não pode ser negativo")
	}
	if c.OverheadPercentage < 0 || c.OverheadPercentage > 100 {
		return errors.New("o percentual de overhead deve estar entre 0 e 100")
	}
	// Profit margin is allowed to exceed 100% (e.g. keystone pricing and up),
	// but is capped to guard against obvious data-entry mistakes.
	if c.ProfitMarginPercentage < 0 || c.ProfitMarginPercentage > 1000 {
		return errors.New("o percentual de margem de lucro deve estar entre 0 e 1000")
	}
	if c.PostProcessingCostPerHour < 0 {
		return errors.New("o custo de pós-processamento por hora não pode ser negativo")
	}
	if c.SupportRemovalCostPerHour < 0 {
		return errors.New("o custo de remoção de suporte por hora não pode ser negativo")
	}
	if c.QualityControlCostPerItem < 0 {
		return errors.New("o custo de controle de qualidade por item não pode ser negativo")
	}
	if c.FailureRatePercentage < 0 || c.FailureRatePercentage > 100 {
		return errors.New("o percentual de taxa de falha deve estar entre 0 e 100")
	}
	if c.WasteGramsPerColorChange < 0 {
		return errors.New("o desperdício por troca de cor não pode ser negativo")
	}

	return nil
}

// CalculateShippingCost calculates total shipping cost based on weight
func (c *CostPresetEntity) CalculateShippingCost(weightInGrams float32) float32 {
	return c.ShippingCostBase + (c.ShippingCostPerGram * weightInGrams)
}

// CalculateOverheadCost calculates overhead based on base cost
func (c *CostPresetEntity) CalculateOverheadCost(baseCost float32) float32 {
	return baseCost * (c.OverheadPercentage / 100)
}

// CalculateProfitMargin calculates profit margin based on total cost
func (c *CostPresetEntity) CalculateProfitMargin(totalCost float32) float32 {
	return totalCost * (c.ProfitMarginPercentage / 100)
}

// CalculateTotalLaborCost calculates total labor cost including post-processing and support removal
func (c *CostPresetEntity) CalculateTotalLaborCost(printTimeHours, postProcessingHours, supportRemovalHours float32) float32 {
	printingCost := c.LaborCostPerHour * printTimeHours
	postProcessingCost := c.PostProcessingCostPerHour * postProcessingHours
	supportRemovalCost := c.SupportRemovalCostPerHour * supportRemovalHours

	return printingCost + postProcessingCost + supportRemovalCost
}
