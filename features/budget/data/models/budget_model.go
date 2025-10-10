package models

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BudgetModel represents the budget data model for GORM
type BudgetModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index:idx_budget_org" json:"organization_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description"`

	// Foreign key to Customer
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`

	// Status
	Status string `gorm:"type:varchar(20);not null" json:"status"`

	// Print time
	PrintTimeHours   int `gorm:"type:integer;not null" json:"print_time_hours"`
	PrintTimeMinutes int `gorm:"type:integer;not null" json:"print_time_minutes"`

	// Presets (foreign keys)
	MachinePresetID *uuid.UUID `gorm:"type:uuid" json:"machine_preset_id"`
	EnergyPresetID  *uuid.UUID `gorm:"type:uuid" json:"energy_preset_id"`
	CostPresetID    *uuid.UUID `gorm:"type:uuid" json:"cost_preset_id"`

	// Configuration flags
	IncludeEnergyCost bool     `gorm:"default:false" json:"include_energy_cost"`
	IncludeLaborCost  bool     `gorm:"default:false" json:"include_labor_cost"`
	IncludeWasteCost  bool     `gorm:"default:false" json:"include_waste_cost"`
	LaborCostPerHour  *float64 `gorm:"type:numeric" json:"labor_cost_per_hour"`

	// Calculated costs (in cents)
	FilamentCost int64 `gorm:"type:bigint;default:0" json:"filament_cost"`
	WasteCost    int64 `gorm:"type:bigint;default:0" json:"waste_cost"`
	EnergyCost   int64 `gorm:"type:bigint;default:0" json:"energy_cost"`
	LaborCost    int64 `gorm:"type:bigint;default:0" json:"labor_cost"`
	TotalCost    int64 `gorm:"type:bigint;default:0" json:"total_cost"`

	// Additional fields for PDF generation
	DeliveryDays *int    `gorm:"type:integer" json:"delivery_days"`
	PaymentTerms *string `gorm:"type:text" json:"payment_terms"`
	Notes        *string `gorm:"type:text" json:"notes"`
	PDFUrl       *string `gorm:"type:varchar(500)" json:"pdf_url"`

	// Ownership
	OwnerUserID string `gorm:"type:varchar(255);not null;index" json:"owner_user_id"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (BudgetModel) TableName() string {
	return "budgets"
}

// BeforeCreate is a GORM hook executed before creating a budget
func (b *BudgetModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.Status == "" {
		b.Status = string(entities.StatusDraft)
	}
	return nil
}

// ToEntity converts the GORM model to domain entity
func (b *BudgetModel) ToEntity() *entities.BudgetEntity {
	return &entities.BudgetEntity{
		ID:                b.ID,
		OrganizationID:    b.OrganizationID,
		Name:              b.Name,
		Description:       b.Description,
		CustomerID:        b.CustomerID,
		Status:            entities.BudgetStatus(b.Status),
		PrintTimeHours:    b.PrintTimeHours,
		PrintTimeMinutes:  b.PrintTimeMinutes,
		MachinePresetID:   b.MachinePresetID,
		EnergyPresetID:    b.EnergyPresetID,
		CostPresetID:      b.CostPresetID,
		IncludeEnergyCost: b.IncludeEnergyCost,
		IncludeLaborCost:  b.IncludeLaborCost,
		IncludeWasteCost:  b.IncludeWasteCost,
		LaborCostPerHour:  b.LaborCostPerHour,
		FilamentCost:      b.FilamentCost,
		WasteCost:         b.WasteCost,
		EnergyCost:        b.EnergyCost,
		LaborCost:         b.LaborCost,
		TotalCost:         b.TotalCost,
		DeliveryDays:      b.DeliveryDays,
		PaymentTerms:      b.PaymentTerms,
		Notes:             b.Notes,
		PDFUrl:            b.PDFUrl,
		OwnerUserID:       b.OwnerUserID,
		CreatedAt:         b.CreatedAt,
		UpdatedAt:         b.UpdatedAt,
		DeletedAt:         getDeletedAt(b.DeletedAt),
	}
}

// getDeletedAt returns nil if deleted_at is not valid, otherwise returns pointer to time
func getDeletedAt(deletedAt gorm.DeletedAt) *time.Time {
	if deletedAt.Valid {
		return &deletedAt.Time
	}
	return nil
}

// FromEntity converts domain entity to GORM model
func (b *BudgetModel) FromEntity(entity *entities.BudgetEntity) {
	b.ID = entity.ID
	b.OrganizationID = entity.OrganizationID
	b.Name = entity.Name
	b.Description = entity.Description
	b.CustomerID = entity.CustomerID
	b.Status = string(entity.Status)
	b.PrintTimeHours = entity.PrintTimeHours
	b.PrintTimeMinutes = entity.PrintTimeMinutes
	b.MachinePresetID = entity.MachinePresetID
	b.EnergyPresetID = entity.EnergyPresetID
	b.CostPresetID = entity.CostPresetID
	b.IncludeEnergyCost = entity.IncludeEnergyCost
	b.IncludeLaborCost = entity.IncludeLaborCost
	b.IncludeWasteCost = entity.IncludeWasteCost
	b.LaborCostPerHour = entity.LaborCostPerHour
	b.FilamentCost = entity.FilamentCost
	b.WasteCost = entity.WasteCost
	b.EnergyCost = entity.EnergyCost
	b.LaborCost = entity.LaborCost
	b.TotalCost = entity.TotalCost
	b.DeliveryDays = entity.DeliveryDays
	b.PaymentTerms = entity.PaymentTerms
	b.Notes = entity.Notes
	b.PDFUrl = entity.PDFUrl
	b.OwnerUserID = entity.OwnerUserID
	b.CreatedAt = entity.CreatedAt
	b.UpdatedAt = entity.UpdatedAt
	if entity.DeletedAt != nil {
		b.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}
	}
}
