// Package models holds the GORM models for filament stock control.
package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/stock/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StockMovementModel is the GORM model for the filament_stock_movements ledger.
//
// Foreign keys (filament_id ON DELETE CASCADE, budget_id ON DELETE SET NULL,
// organization_id ON DELETE RESTRICT) and the (org, filament, created_at desc)
// and partial-unique (budget_id, filament_id) WHERE type='consumption' indexes are
// created idempotently in RunMigrations, since AutoMigrate FK creation is disabled.
type StockMovementModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"`
	FilamentID     uuid.UUID `gorm:"type:uuid;not null;index" json:"filament_id"`
	Type           string    `gorm:"type:varchar(20);not null" json:"type"`
	// Grams is the signed delta (negative for waste/consumption).
	Grams int64 `gorm:"type:bigint;not null" json:"grams"`
	// UnitPricePerKg is cents/kg, purchase movements only (nullable).
	UnitPricePerKg *int64     `gorm:"type:bigint" json:"unit_price_per_kg"`
	BudgetID       *uuid.UUID `gorm:"type:uuid;index" json:"budget_id"`
	Note           *string    `gorm:"type:varchar(500)" json:"note"`
	CreatedBy      string     `gorm:"type:varchar(255);not null" json:"created_by"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for GORM.
func (StockMovementModel) TableName() string {
	return "filament_stock_movements"
}

// BeforeCreate assigns an ID when absent.
func (m *StockMovementModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to a domain entity.
func (m *StockMovementModel) ToEntity() *entities.StockMovementEntity {
	return &entities.StockMovementEntity{
		ID:             m.ID,
		OrganizationID: m.OrganizationID,
		FilamentID:     m.FilamentID,
		Type:           entities.MovementType(m.Type),
		Grams:          m.Grams,
		UnitPricePerKg: m.UnitPricePerKg,
		BudgetID:       m.BudgetID,
		Note:           m.Note,
		CreatedBy:      m.CreatedBy,
		CreatedAt:      m.CreatedAt,
	}
}
