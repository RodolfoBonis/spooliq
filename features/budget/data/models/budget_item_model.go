package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetItemModel represents the budget item data model for GORM
type BudgetItemModel struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetID   uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_id"`
	FilamentID uuid.UUID `gorm:"type:uuid;not null" json:"filament_id"`

	// Quantity and order
	Quantity float64 `gorm:"type:numeric;not null" json:"quantity"` // grams
	Order    int     `gorm:"type:integer;not null" json:"order"`    // sequence

	// Calculated values
	WasteAmount float64 `gorm:"type:numeric;default:0" json:"waste_amount"` // grams
	ItemCost    int64   `gorm:"type:bigint;default:0" json:"item_cost"`     // cents

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
		ID:          bi.ID,
		BudgetID:    bi.BudgetID,
		FilamentID:  bi.FilamentID,
		Quantity:    bi.Quantity,
		Order:       bi.Order,
		WasteAmount: bi.WasteAmount,
		ItemCost:    bi.ItemCost,
		CreatedAt:   bi.CreatedAt,
		UpdatedAt:   bi.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (bi *BudgetItemModel) FromEntity(entity *entities.BudgetItemEntity) {
	bi.ID = entity.ID
	bi.BudgetID = entity.BudgetID
	bi.FilamentID = entity.FilamentID
	bi.Quantity = entity.Quantity
	bi.Order = entity.Order
	bi.WasteAmount = entity.WasteAmount
	bi.ItemCost = entity.ItemCost
	bi.CreatedAt = entity.CreatedAt
	bi.UpdatedAt = entity.UpdatedAt
}
