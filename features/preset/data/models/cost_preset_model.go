package models

import (
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/google/uuid"
)

// CostPresetModel represents fixed costs preset in the database
type CostPresetModel struct {
	ID                        uuid.UUID `gorm:"<-:create;type:uuid;primaryKey" json:"id"`
	LaborCostPerHour          float32   `gorm:"type:float" json:"labor_cost_per_hour"`
	PackagingCostPerItem      float32   `gorm:"type:float" json:"packaging_cost_per_item"`
	ShippingCostBase          float32   `gorm:"type:float" json:"shipping_cost_base"`
	ShippingCostPerGram       float32   `gorm:"type:float" json:"shipping_cost_per_gram"`
	OverheadPercentage        float32   `gorm:"type:float" json:"overhead_percentage"`
	ProfitMarginPercentage    float32   `gorm:"type:float" json:"profit_margin_percentage"`
	PostProcessingCostPerHour float32   `gorm:"type:float" json:"post_processing_cost_per_hour"`
	SupportRemovalCostPerHour float32   `gorm:"type:float" json:"support_removal_cost_per_hour"`
	QualityControlCostPerItem float32   `gorm:"type:float" json:"quality_control_cost_per_item"`
}

// TableName returns the table name for the cost preset model
func (c *CostPresetModel) TableName() string { return "cost_presets" }

// FromEntity populates the CostPresetModel from a CostPresetEntity
func (c *CostPresetModel) FromEntity(entity *entities.CostPresetEntity) {
	c.ID = entity.ID
	c.LaborCostPerHour = entity.LaborCostPerHour
	c.PackagingCostPerItem = entity.PackagingCostPerItem
	c.ShippingCostBase = entity.ShippingCostBase
	c.ShippingCostPerGram = entity.ShippingCostPerGram
	c.OverheadPercentage = entity.OverheadPercentage
	c.ProfitMarginPercentage = entity.ProfitMarginPercentage
	c.PostProcessingCostPerHour = entity.PostProcessingCostPerHour
	c.SupportRemovalCostPerHour = entity.SupportRemovalCostPerHour
	c.QualityControlCostPerItem = entity.QualityControlCostPerItem
}

// ToEntity converts the CostPresetModel to a CostPresetEntity
func (c *CostPresetModel) ToEntity() entities.CostPresetEntity {
	return entities.CostPresetEntity{
		ID:                        c.ID,
		LaborCostPerHour:          c.LaborCostPerHour,
		PackagingCostPerItem:      c.PackagingCostPerItem,
		ShippingCostBase:          c.ShippingCostBase,
		ShippingCostPerGram:       c.ShippingCostPerGram,
		OverheadPercentage:        c.OverheadPercentage,
		ProfitMarginPercentage:    c.ProfitMarginPercentage,
		PostProcessingCostPerHour: c.PostProcessingCostPerHour,
		SupportRemovalCostPerHour: c.SupportRemovalCostPerHour,
		QualityControlCostPerItem: c.QualityControlCostPerItem,
	}
}
