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
}

// Validate validates the cost preset entity
func (c *CostPresetEntity) Validate() error {
	if c.LaborCostPerHour < 0 {
		return errors.New("labor cost per hour cannot be negative")
	}
	if c.PackagingCostPerItem < 0 {
		return errors.New("packaging cost per item cannot be negative")
	}
	if c.ShippingCostBase < 0 {
		return errors.New("shipping cost base cannot be negative")
	}
	if c.ShippingCostPerGram < 0 {
		return errors.New("shipping cost per gram cannot be negative")
	}
	if c.OverheadPercentage < 0 || c.OverheadPercentage > 100 {
		return errors.New("overhead percentage must be between 0 and 100")
	}
	if c.ProfitMarginPercentage < 0 || c.ProfitMarginPercentage > 100 {
		return errors.New("profit margin percentage must be between 0 and 100")
	}
	if c.PostProcessingCostPerHour < 0 {
		return errors.New("post processing cost per hour cannot be negative")
	}
	if c.SupportRemovalCostPerHour < 0 {
		return errors.New("support removal cost per hour cannot be negative")
	}
	if c.QualityControlCostPerItem < 0 {
		return errors.New("quality control cost per item cannot be negative")
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
