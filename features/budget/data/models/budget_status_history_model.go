package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetStatusHistoryModel represents the budget status history data model for GORM
type BudgetStatusHistoryModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_id"`
	PreviousStatus string    `gorm:"type:varchar(20);not null" json:"previous_status"`
	NewStatus      string    `gorm:"type:varchar(20);not null" json:"new_status"`
	ChangedBy      string    `gorm:"type:varchar(255);not null" json:"changed_by"`
	Notes          string    `gorm:"type:text" json:"notes"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for GORM
func (BudgetStatusHistoryModel) TableName() string {
	return "budget_status_history"
}

// BeforeCreate is a GORM hook executed before creating a history entry
func (bh *BudgetStatusHistoryModel) BeforeCreate(tx *gorm.DB) error {
	if bh.ID == uuid.Nil {
		bh.ID = uuid.New()
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (bh *BudgetStatusHistoryModel) ToEntity() *entities.BudgetStatusHistoryEntity {
	return &entities.BudgetStatusHistoryEntity{
		ID:             bh.ID,
		BudgetID:       bh.BudgetID,
		PreviousStatus: entities.BudgetStatus(bh.PreviousStatus),
		NewStatus:      entities.BudgetStatus(bh.NewStatus),
		ChangedBy:      bh.ChangedBy,
		Notes:          bh.Notes,
		CreatedAt:      bh.CreatedAt,
	}
}

// FromEntity converts domain entity to GORM model
func (bh *BudgetStatusHistoryModel) FromEntity(entity *entities.BudgetStatusHistoryEntity) {
	bh.ID = entity.ID
	bh.BudgetID = entity.BudgetID
	bh.PreviousStatus = string(entity.PreviousStatus)
	bh.NewStatus = string(entity.NewStatus)
	bh.ChangedBy = entity.ChangedBy
	bh.Notes = entity.Notes
	bh.CreatedAt = entity.CreatedAt
}
