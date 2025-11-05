package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	companyModels "github.com/RodolfoBonis/spooliq/features/company/data/models"
	filamentModels "github.com/RodolfoBonis/spooliq/features/filament/data/models"
	"github.com/google/uuid"
)

// BudgetItemFilamentModel represents the relationship between budget items and filaments
type BudgetItemFilamentModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetItemID   uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_item_id"`
	FilamentID     uuid.UUID `gorm:"type:uuid;not null;index" json:"filament_id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"` // FK: references companies(organization_id) ON DELETE RESTRICT

	// Quantidade TOTAL de filamento para este item (não por unidade!)
	// Exemplo: Para imprimir 100 chaveiros em lote, usar 2800g de PLA Rosa
	// (economias de escala, menos desperdício, melhor aproveitamento)
	Quantity float64 `gorm:"type:numeric;not null" json:"quantity"` // gramas TOTAL

	// Ordem de aplicação (para AMS/multi-cor)
	Order int `gorm:"type:integer;not null;default:1" json:"order"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// GORM v2 Relationships
	Organization *companyModels.CompanyModel   `gorm:"foreignKey:OrganizationID;references:OrganizationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"organization,omitempty"`
	BudgetItem   *BudgetItemModel              `gorm:"foreignKey:BudgetItemID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"budget_item,omitempty"`
	Filament     *filamentModels.FilamentModel `gorm:"foreignKey:FilamentID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"filament,omitempty"`
}

// TableName returns the database table name for budget item filaments
func (BudgetItemFilamentModel) TableName() string {
	return "budget_item_filaments"
}

// ToEntity converts model to entity
func (m *BudgetItemFilamentModel) ToEntity() *entities.BudgetItemFilamentEntity {
	return &entities.BudgetItemFilamentEntity{
		ID:             m.ID,
		BudgetItemID:   m.BudgetItemID,
		FilamentID:     m.FilamentID,
		OrganizationID: m.OrganizationID,
		Quantity:       m.Quantity,
		Order:          m.Order,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// FromEntity converts entity to model
func (m *BudgetItemFilamentModel) FromEntity(e *entities.BudgetItemFilamentEntity) {
	m.ID = e.ID
	m.BudgetItemID = e.BudgetItemID
	m.FilamentID = e.FilamentID
	m.OrganizationID = e.OrganizationID
	m.Quantity = e.Quantity
	m.Order = e.Order
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
}
