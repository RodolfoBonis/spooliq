package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	supplyModels "github.com/RodolfoBonis/spooliq/features/supply/data/models"
	"github.com/google/uuid"
)

// BudgetItemSupplyModel represents a non-printed supply used in a budget item.
//
// Name, Unit and UnitCost are a snapshot of the catalog supply at the last
// calculation (so the budget stays reproducible independent of later catalog
// edits/deletions). The supply FK uses ON DELETE RESTRICT: a supply still
// referenced by a budget item cannot be hard-deleted, but soft-deleting it in the
// catalog is always allowed and does not affect stored budgets.
type BudgetItemSupplyModel struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetItemID     uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_item_id"`
	SupplyID         uuid.UUID `gorm:"type:uuid;not null;index" json:"supply_id"`
	OrganizationID   string    `gorm:"type:varchar(255);not null;index" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT
	QuantityPerPiece float64   `gorm:"type:numeric;not null;default:0" json:"quantity_per_piece"`
	Order            int       `gorm:"type:integer;not null;default:0" json:"order"`

	// Snapshot of the catalog supply at the last calculation.
	Name string `gorm:"type:varchar(255);not null" json:"name"`
	Unit string `gorm:"type:varchar(20);not null" json:"unit"`
	// PackageSize/PackagePrice snapshot the supply's package pricing; the line cost is
	// round(quantity_per_piece * product_quantity * package_price / package_size).
	// PackageSize carries a gorm default of 1 so AutoMigrate backfills legacy rows
	// with a single-unit package (the data migration seeds package_price from unit_cost).
	PackageSize  float64 `gorm:"type:numeric;not null;default:1" json:"package_size"` // units per package, > 0
	PackagePrice int64   `gorm:"type:bigint;not null;default:0" json:"package_price"` // cents
	// UnitCost is the DERIVED per-unit cost (round(package_price / package_size)), kept
	// for readers predating package pricing.
	UnitCost int64 `gorm:"type:bigint;not null;default:0" json:"unit_cost"` // cents

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	BudgetItem   *BudgetItemModel            `gorm:"foreignKey:BudgetItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"budget_item,omitempty"`
	Supply       *supplyModels.SupplyModel   `gorm:"foreignKey:SupplyID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"supply,omitempty"`
}

// TableName returns the database table name for budget item supplies.
func (BudgetItemSupplyModel) TableName() string {
	return "budget_item_supplies"
}

// ToEntity converts the model to a domain entity.
func (m *BudgetItemSupplyModel) ToEntity() *entities.BudgetItemSupplyEntity {
	return &entities.BudgetItemSupplyEntity{
		ID:               m.ID,
		BudgetItemID:     m.BudgetItemID,
		SupplyID:         m.SupplyID,
		OrganizationID:   m.OrganizationID,
		QuantityPerPiece: m.QuantityPerPiece,
		Order:            m.Order,
		Name:             m.Name,
		Unit:             m.Unit,
		PackageSize:      m.PackageSize,
		PackagePrice:     m.PackagePrice,
		UnitCost:         m.UnitCost,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

// FromEntity populates the model from a domain entity.
func (m *BudgetItemSupplyModel) FromEntity(e *entities.BudgetItemSupplyEntity) {
	m.ID = e.ID
	m.BudgetItemID = e.BudgetItemID
	m.SupplyID = e.SupplyID
	m.OrganizationID = e.OrganizationID
	m.QuantityPerPiece = e.QuantityPerPiece
	m.Order = e.Order
	m.Name = e.Name
	m.Unit = e.Unit
	m.PackageSize = e.PackageSize
	m.PackagePrice = e.PackagePrice
	m.UnitCost = e.UnitCost
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
}
