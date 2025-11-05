package models

import (
	"encoding/json"
	"fmt"
	"time"

	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlanTemplateModel represents the plan template data model for GORM
type PlanTemplateModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Category    string    `gorm:"type:varchar(100);not null;index" json:"category"`
	PlanData    string    `gorm:"type:jsonb;not null" json:"plan_data"` // JSON string
	IsActive    bool      `gorm:"not null;default:true;index" json:"is_active"`
	UsageCount  int       `gorm:"not null;default:0" json:"usage_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName specifies the table name for GORM
func (PlanTemplateModel) TableName() string {
	return "plan_templates"
}

// ToEntity converts model to entity
func (m *PlanTemplateModel) ToEntity() *adminEntities.PlanTemplate {
	var planData adminEntities.PlanTemplateData
	if m.PlanData != "" {
		json.Unmarshal([]byte(m.PlanData), &planData)
	}

	return &adminEntities.PlanTemplate{
		ID:          m.ID.String(),
		Name:        m.Name,
		Description: m.Description,
		Category:    m.Category,
		PlanData:    planData,
		IsActive:    m.IsActive,
		UsageCount:  m.UsageCount,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// FromEntity converts entity to model
func (m *PlanTemplateModel) FromEntity(entity *adminEntities.PlanTemplate) {
	if entity.ID != "" {
		m.ID = uuid.MustParse(entity.ID)
	}
	m.Name = entity.Name
	m.Description = entity.Description
	m.Category = entity.Category
	m.IsActive = entity.IsActive
	m.UsageCount = entity.UsageCount
	m.CreatedAt = entity.CreatedAt
	m.UpdatedAt = entity.UpdatedAt

	planDataBytes, _ := json.Marshal(entity.PlanData)
	m.PlanData = string(planDataBytes)
}

// BeforeCreate hook for GORM
func (m *PlanTemplateModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// PlanMigrationModel represents the plan migration data model for GORM
type PlanMigrationModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FromPlanID      uuid.UUID `gorm:"type:uuid;not null;index" json:"from_plan_id"`
	ToPlanID        uuid.UUID `gorm:"type:uuid;not null;index" json:"to_plan_id"`
	Status          string    `gorm:"type:varchar(50);not null;index" json:"status"`
	TotalCompanies  int       `gorm:"not null;default:0" json:"total_companies"`
	Successful      int       `gorm:"not null;default:0" json:"successful"`
	Failed          int       `gorm:"not null;default:0" json:"failed"`
	Results         string    `gorm:"type:jsonb" json:"results"` // JSON string
	Reason          string    `gorm:"type:text;not null" json:"reason"`
	UserID          string    `gorm:"type:varchar(255);not null" json:"user_id"`
	UserEmail       string    `gorm:"type:varchar(255);not null" json:"user_email"`
	ScheduledFor    *time.Time `gorm:"type:timestamp" json:"scheduled_for"`
	CompletedAt     *time.Time `gorm:"type:timestamp" json:"completed_at"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	FromPlan *SubscriptionPlanModel `gorm:"foreignKey:FromPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"from_plan,omitempty"`
	ToPlan   *SubscriptionPlanModel `gorm:"foreignKey:ToPlanID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"to_plan,omitempty"`
}

// TableName specifies the table name for GORM
func (PlanMigrationModel) TableName() string {
	return "plan_migrations"
}

// ToEntity converts model to entity
func (m *PlanMigrationModel) ToEntity() *adminEntities.PlanMigrationResult {
	var results []adminEntities.MigrationCompanyItem
	if m.Results != "" {
		json.Unmarshal([]byte(m.Results), &results)
	}

	fromPlanName := ""
	toPlanName := ""
	if m.FromPlan != nil {
		fromPlanName = m.FromPlan.Name
	}
	if m.ToPlan != nil {
		toPlanName = m.ToPlan.Name
	}

	return &adminEntities.PlanMigrationResult{
		MigrationID:     m.ID.String(),
		FromPlanID:      m.FromPlanID.String(),
		FromPlanName:    fromPlanName,
		ToPlanID:        m.ToPlanID.String(),
		ToPlanName:      toPlanName,
		TotalCompanies:  m.TotalCompanies,
		Successful:      m.Successful,
		Failed:          m.Failed,
		Results:         results,
		Status:          m.Status,
		ScheduledFor:    m.ScheduledFor,
		CompletedAt:     m.CompletedAt,
		Summary:         m.generateSummary(),
	}
}

// generateSummary generates a human-readable summary
func (m *PlanMigrationModel) generateSummary() string {
	if m.Status == "completed" {
		return fmt.Sprintf("Migration completed: %d/%d companies successfully migrated from %s to %s", 
			m.Successful, m.TotalCompanies, m.FromPlan.Name, m.ToPlan.Name)
	}
	return fmt.Sprintf("Migration %s: %d companies to be migrated", m.Status, m.TotalCompanies)
}

// BeforeCreate hook for GORM
func (m *PlanMigrationModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}