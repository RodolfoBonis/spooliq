package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetItemModel represents the budget item data model for GORM
type BudgetItemModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_id"`
	FilamentID     uuid.UUID `gorm:"type:uuid;not null" json:"filament_id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"`

	// Filament quantity (internal - for cost calculation)
	Quantity float64 `gorm:"type:numeric;not null" json:"quantity"` // grams
	Order    int     `gorm:"type:integer;not null" json:"order"`    // sequence

	// Product information (customer-facing - for PDF and quotes)
	ProductName        string  `gorm:"type:varchar(255);not null" json:"product_name"`
	ProductDescription *string `gorm:"type:text" json:"product_description,omitempty"`
	ProductQuantity    int     `gorm:"type:integer;not null" json:"product_quantity"` // number of units
	UnitPrice          int64   `gorm:"type:bigint;not null" json:"unit_price"`        // cents per unit
	ProductDimensions  *string `gorm:"type:varchar(100)" json:"product_dimensions,omitempty"`

	// NEW: Print time for THIS item (not global)
	PrintTimeHours   int `gorm:"type:integer;default:0" json:"print_time_hours"`
	PrintTimeMinutes int `gorm:"type:integer;default:0" json:"print_time_minutes"`

	// NEW: Additional costs specific to this item
	CostPresetID        *uuid.UUID `gorm:"type:uuid" json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64     `gorm:"type:bigint" json:"additional_labor_cost,omitempty"` // cents (pintura, acabamento, etc)
	AdditionalNotes     *string    `gorm:"type:text" json:"additional_notes,omitempty"`

	// NEW: Calculated costs per item
	FilamentCost  int64 `gorm:"type:bigint;default:0" json:"filament_cost"`   // cents
	WasteCost     int64 `gorm:"type:bigint;default:0" json:"waste_cost"`      // cents
	EnergyCost    int64 `gorm:"type:bigint;default:0" json:"energy_cost"`     // cents
	LaborCost     int64 `gorm:"type:bigint;default:0" json:"labor_cost"`      // cents
	ItemTotalCost int64 `gorm:"type:bigint;default:0" json:"item_total_cost"` // cents (sum of all costs)

	// OLD FIELDS (will be removed in later migration, kept for compatibility)
	WasteAmount float64 `gorm:"type:numeric;default:0" json:"waste_amount"` // grams (deprecated)
	ItemCost    int64   `gorm:"type:bigint;default:0" json:"item_cost"`     // cents (deprecated, use ItemTotalCost)

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (BudgetItemModel) TableName() string {
	return "budget_items"
}

// BeforeCreate is a GORM hook executed before creating a budget item
func (bi *BudgetItemModel) BeforeCreate(tx *gorm.DB) error {
	if bi.ID == uuid.Nil {
		bi.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (bi *BudgetItemModel) ToEntity() *entities.BudgetItemEntity {
	return &entities.BudgetItemEntity{
		ID:                 bi.ID,
		BudgetID:           bi.BudgetID,
		FilamentID:         bi.FilamentID,
		Quantity:           bi.Quantity,
		Order:              bi.Order,
		ProductName:        bi.ProductName,
		ProductDescription: bi.ProductDescription,
		ProductQuantity:    bi.ProductQuantity,
		UnitPrice:          bi.UnitPrice,
		ProductDimensions:  bi.ProductDimensions,
		PrintTimeHours:     bi.PrintTimeHours,
		PrintTimeMinutes:   bi.PrintTimeMinutes,
		CostPresetID:       bi.CostPresetID,
		AdditionalLaborCost: bi.AdditionalLaborCost,
		AdditionalNotes:    bi.AdditionalNotes,
		FilamentCost:       bi.FilamentCost,
		WasteCost:          bi.WasteCost,
		EnergyCost:         bi.EnergyCost,
		LaborCost:          bi.LaborCost,
		ItemTotalCost:      bi.ItemTotalCost,
		WasteAmount:        bi.WasteAmount,
		ItemCost:           bi.ItemCost,
		CreatedAt:          bi.CreatedAt,
		UpdatedAt:          bi.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (bi *BudgetItemModel) FromEntity(entity *entities.BudgetItemEntity) {
	bi.ID = entity.ID
	bi.BudgetID = entity.BudgetID
	bi.FilamentID = entity.FilamentID
	bi.Quantity = entity.Quantity
	bi.Order = entity.Order
	bi.ProductName = entity.ProductName
	bi.ProductDescription = entity.ProductDescription
	bi.ProductQuantity = entity.ProductQuantity
	bi.UnitPrice = entity.UnitPrice
	bi.ProductDimensions = entity.ProductDimensions
	bi.PrintTimeHours = entity.PrintTimeHours
	bi.PrintTimeMinutes = entity.PrintTimeMinutes
	bi.CostPresetID = entity.CostPresetID
	bi.AdditionalLaborCost = entity.AdditionalLaborCost
	bi.AdditionalNotes = entity.AdditionalNotes
	bi.FilamentCost = entity.FilamentCost
	bi.WasteCost = entity.WasteCost
	bi.EnergyCost = entity.EnergyCost
	bi.LaborCost = entity.LaborCost
	bi.ItemTotalCost = entity.ItemTotalCost
	bi.WasteAmount = entity.WasteAmount
	bi.ItemCost = entity.ItemCost
	bi.CreatedAt = entity.CreatedAt
	bi.UpdatedAt = entity.UpdatedAt
}
