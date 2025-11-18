package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	filamentModels "github.com/RodolfoBonis/spooliq/features/filament/data/models"
	presetModels "github.com/RodolfoBonis/spooliq/features/preset/data/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetItemModel represents the budget item data model for GORM
type BudgetItemModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_id"`
	FilamentID     uuid.UUID `gorm:"type:uuid;not null" json:"filament_id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT

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

	// Labor breakdown fields
	SetupTimeMinutes        int `gorm:"type:integer;default:0" json:"setup_time_minutes"`         // Setup time for this product (minutes)
	ManualLaborMinutesTotal int `gorm:"type:integer;default:0" json:"manual_labor_minutes_total"` // Total manual labor time for ALL units (minutes)

	// Additional costs specific to this item
	CostPresetID    *uuid.UUID `gorm:"type:uuid" json:"cost_preset_id,omitempty"`
	AdditionalNotes *string    `gorm:"type:text" json:"additional_notes,omitempty"`

	// Calculated costs per item
	FilamentCost    int64 `gorm:"type:bigint;default:0" json:"filament_cost"`     // cents
	WasteCost       int64 `gorm:"type:bigint;default:0" json:"waste_cost"`        // cents
	EnergyCost      int64 `gorm:"type:bigint;default:0" json:"energy_cost"`       // cents
	SetupCost       int64 `gorm:"type:bigint;default:0" json:"setup_cost"`        // cents
	ManualLaborCost int64 `gorm:"type:bigint;default:0" json:"manual_labor_cost"` // cents
	ItemTotalCost   int64 `gorm:"type:bigint;default:0" json:"item_total_cost"`   // cents (sum of all costs)

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel   `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	Budget       *BudgetModel                  `gorm:"foreignKey:BudgetID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"budget,omitempty"`
	Filament     *filamentModels.FilamentModel `gorm:"foreignKey:FilamentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"filament,omitempty"`
	CostPreset   *presetModels.CostPresetModel `gorm:"foreignKey:CostPresetID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"cost_preset,omitempty"`
	Filaments    []BudgetItemFilamentModel     `gorm:"foreignKey:BudgetItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"filaments,omitempty"`
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
		ID:                      bi.ID,
		BudgetID:                bi.BudgetID,
		FilamentID:              bi.FilamentID,
		OrganizationID:          bi.OrganizationID,
		Quantity:                bi.Quantity,
		Order:                   bi.Order,
		ProductName:             bi.ProductName,
		ProductDescription:      bi.ProductDescription,
		ProductQuantity:         bi.ProductQuantity,
		UnitPrice:               bi.UnitPrice,
		ProductDimensions:       bi.ProductDimensions,
		PrintTimeHours:          bi.PrintTimeHours,
		PrintTimeMinutes:        bi.PrintTimeMinutes,
		SetupTimeMinutes:        bi.SetupTimeMinutes,
		ManualLaborMinutesTotal: bi.ManualLaborMinutesTotal,
		CostPresetID:            bi.CostPresetID,
		AdditionalNotes:         bi.AdditionalNotes,
		FilamentCost:            bi.FilamentCost,
		WasteCost:               bi.WasteCost,
		EnergyCost:              bi.EnergyCost,
		SetupCost:               bi.SetupCost,
		ManualLaborCost:         bi.ManualLaborCost,
		ItemTotalCost:           bi.ItemTotalCost,
		CreatedAt:               bi.CreatedAt,
		UpdatedAt:               bi.UpdatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (bi *BudgetItemModel) FromEntity(entity *entities.BudgetItemEntity) {
	bi.ID = entity.ID
	bi.BudgetID = entity.BudgetID
	bi.FilamentID = entity.FilamentID
	bi.OrganizationID = entity.OrganizationID
	bi.Quantity = entity.Quantity
	bi.Order = entity.Order
	bi.ProductName = entity.ProductName
	bi.ProductDescription = entity.ProductDescription
	bi.ProductQuantity = entity.ProductQuantity
	bi.UnitPrice = entity.UnitPrice
	bi.ProductDimensions = entity.ProductDimensions
	bi.PrintTimeHours = entity.PrintTimeHours
	bi.PrintTimeMinutes = entity.PrintTimeMinutes
	bi.SetupTimeMinutes = entity.SetupTimeMinutes
	bi.ManualLaborMinutesTotal = entity.ManualLaborMinutesTotal
	bi.CostPresetID = entity.CostPresetID
	bi.AdditionalNotes = entity.AdditionalNotes
	bi.FilamentCost = entity.FilamentCost
	bi.WasteCost = entity.WasteCost
	bi.EnergyCost = entity.EnergyCost
	bi.SetupCost = entity.SetupCost
	bi.ManualLaborCost = entity.ManualLaborCost
	bi.ItemTotalCost = entity.ItemTotalCost
	bi.CreatedAt = entity.CreatedAt
	bi.UpdatedAt = entity.UpdatedAt
}
